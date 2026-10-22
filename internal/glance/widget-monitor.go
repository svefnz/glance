package glance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	monitorWidgetTemplate        = mustParseTemplate("monitor.html", "widget-base.html")
	monitorWidgetCompactTemplate = mustParseTemplate("monitor-compact.html", "widget-base.html")
)

type monitorWidget struct {
	widgetBase `yaml:",inline"`
	Sites      []struct {
		*SiteStatusRequest `yaml:",inline"`
		Status             *siteStatus     `yaml:"-"`
		URL                string          `yaml:"-"`
		ErrorURL           string          `yaml:"error-url"`
		Title              string          `yaml:"title"`
		Icon               customIconField `yaml:"icon"`
		SameTab            bool            `yaml:"same-tab"`
		StatusText         string          `yaml:"-"`
		StatusStyle        string          `yaml:"-"`
		AltStatusCodes     []int           `yaml:"alt-status-codes"`

		// Container optionally names the container backing this site. When
		// agent-url is set, its usage is shown alongside the site status.
		Container     string  `yaml:"container"`
		CPUPercent    float64 `yaml:"-"`
		MemoryUsedMB  uint64  `yaml:"-"`
		MemoryLimitMB uint64  `yaml:"-"`
		HasUsage      bool    `yaml:"-"`
	} `yaml:"sites"`
	Style           string `yaml:"style"`
	ShowFailingOnly bool   `yaml:"show-failing-only"`
	AgentURL        string `yaml:"agent-url"`
	HasFailing      bool   `yaml:"-"`
}

func (widget *monitorWidget) initialize() error {
	widget.withTitle("Monitor").withCacheDuration(5 * time.Minute)

	return nil
}

func (widget *monitorWidget) update(ctx context.Context) {
	requests := make([]*SiteStatusRequest, len(widget.Sites))

	for i := range widget.Sites {
		requests[i] = widget.Sites[i].SiteStatusRequest
	}

	statuses, err := fetchStatusForSites(requests)

	if !widget.canContinueUpdateAfterHandlingErr(err) {
		return
	}

	widget.HasFailing = false

	for i := range widget.Sites {
		site := &widget.Sites[i]
		status := &statuses[i]
		site.Status = status

		if !slices.Contains(site.AltStatusCodes, status.Code) && (status.Code >= 400 || status.Error != nil) {
			widget.HasFailing = true
		}

		if status.Error != nil && site.ErrorURL != "" {
			site.URL = site.ErrorURL
		} else {
			site.URL = site.DefaultURL
		}

		site.StatusText = statusCodeToText(status.Code, site.AltStatusCodes)
		site.StatusStyle = statusCodeToStyle(status.Code, site.AltStatusCodes)
	}

	widget.attachContainerUsage()
}

// attachContainerUsage fills in CPU and memory for sites that name a container
// and have an agent configured. Failures leave usage blank rather than failing
// the widget, since the status checks already succeeded.
func (widget *monitorWidget) attachContainerUsage() {
	if widget.AgentURL == "" {
		return
	}

	hasContainer := false
	for i := range widget.Sites {
		if widget.Sites[i].Container != "" {
			hasContainer = true
			break
		}
	}
	if !hasContainer {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	usage, err := fetchAgentContainerUsage(ctx, widget.AgentURL)
	if err != nil {
		slog.Warn("Getting container usage for monitor", "error", err)
		return
	}

	for i := range widget.Sites {
		site := &widget.Sites[i]
		if site.Container == "" {
			continue
		}

		entry, ok := usage[site.Container]
		if !ok {
			continue
		}

		site.CPUPercent = entry.CPUPercent
		site.MemoryUsedMB = entry.MemoryUsedMB
		site.MemoryLimitMB = entry.MemoryLimitMB
		site.HasUsage = entry.MemoryUsedMB > 0
	}
}

func (widget *monitorWidget) Render() template.HTML {
	if widget.Style == "compact" {
		return widget.renderTemplate(widget, monitorWidgetCompactTemplate)
	}

	return widget.renderTemplate(widget, monitorWidgetTemplate)
}

func statusCodeToText(status int, altStatusCodes []int) string {
	if status == 200 || slices.Contains(altStatusCodes, status) {
		return "OK"
	}
	if status == 404 {
		return "Not Found"
	}
	if status == 403 {
		return "Forbidden"
	}
	if status == 401 {
		return "Unauthorized"
	}
	if status >= 500 {
		return "Server Error"
	}
	if status >= 400 {
		return "Client Error"
	}

	return strconv.Itoa(status)
}

func statusCodeToStyle(status int, altStatusCodes []int) string {
	if status == 200 || slices.Contains(altStatusCodes, status) {
		return "ok"
	}

	return "error"
}

type SiteStatusRequest struct {
	DefaultURL    string        `yaml:"url"`
	CheckURL      string        `yaml:"check-url"`
	AllowInsecure bool          `yaml:"allow-insecure"`
	Timeout       durationField `yaml:"timeout"`
	BasicAuth     struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"basic-auth"`
}

type siteStatus struct {
	Code         int
	TimedOut     bool
	ResponseTime time.Duration
	Error        error
}

func fetchSiteStatusTask(statusRequest *SiteStatusRequest) (siteStatus, error) {
	var url string
	if statusRequest.CheckURL != "" {
		url = statusRequest.CheckURL
	} else {
		url = statusRequest.DefaultURL
	}

	timeout := ternary(statusRequest.Timeout > 0, time.Duration(statusRequest.Timeout), 3*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return siteStatus{
			Error: err,
		}, nil
	}

	if statusRequest.BasicAuth.Username != "" || statusRequest.BasicAuth.Password != "" {
		request.SetBasicAuth(statusRequest.BasicAuth.Username, statusRequest.BasicAuth.Password)
	}

	requestSentAt := time.Now()
	var response *http.Response

	if !statusRequest.AllowInsecure {
		response, err = defaultHTTPClient.Do(request)
	} else {
		response, err = defaultInsecureHTTPClient.Do(request)
	}

	status := siteStatus{ResponseTime: time.Since(requestSentAt)}

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			status.TimedOut = true
		}

		status.Error = err
		return status, nil
	}

	defer response.Body.Close()

	status.Code = response.StatusCode

	return status, nil
}

func fetchStatusForSites(requests []*SiteStatusRequest) ([]siteStatus, error) {
	job := newJob(fetchSiteStatusTask, requests).withWorkers(20)
	results, _, err := workerPoolDo(job)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// fetchAgentContainerUsage reads container usage from a glance-agent and keys
// it by container name so sites can look theirs up.
func fetchAgentContainerUsage(ctx context.Context, agentURL string) (map[string]agentContainerStats, error) {
	endpoint := strings.TrimRight(agentURL, "/") + "/api/containers"

	request, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	response, err := defaultHTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("requesting agent: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent returned status %s", response.Status)
	}

	var stats []agentContainerStats
	if err := json.NewDecoder(response.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decoding agent response: %w", err)
	}

	byName := make(map[string]agentContainerStats, len(stats))
	for i := range stats {
		byName[stats[i].Name] = stats[i]
	}

	return byName, nil
}
