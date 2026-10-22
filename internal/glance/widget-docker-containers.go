package glance

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

var dockerContainersWidgetTemplate = mustParseTemplate("docker-containers.html", "widget-base.html")

type dockerContainersWidget struct {
	widgetBase           `yaml:",inline"`
	HideByDefault        bool                         `yaml:"hide-by-default"`
	RunningOnly          bool                         `yaml:"running-only"`
	Category             string                       `yaml:"category"`
	SockPath             string                       `yaml:"sock-path"`
	AgentURL             string                       `yaml:"agent-url"`
	FormatContainerNames bool                         `yaml:"format-container-names"`
	Containers           dockerContainerList          `yaml:"-"`
	LabelOverrides       map[string]map[string]string `yaml:"containers"`
}

func (widget *dockerContainersWidget) initialize() error {
	widget.withTitle("Docker Containers").withCacheDuration(1 * time.Minute)

	if widget.SockPath == "" {
		widget.SockPath = "/var/run/docker.sock"
	}

	return nil
}

func (widget *dockerContainersWidget) update(ctx context.Context) {
	if widget.AgentURL != "" {
		containers, err := fetchContainersFromAgent(widget.AgentURL, widget.LabelOverrides)
		if !widget.canContinueUpdateAfterHandlingErr(err) {
			return
		}

		containers.sortByStateIconThenName()
		widget.Containers = containers
		return
	}

	containers, err := fetchDockerContainers(
		widget.SockPath,
		widget.HideByDefault,
		widget.Category,
		widget.RunningOnly,
		widget.FormatContainerNames,
		widget.LabelOverrides,
	)
	if !widget.canContinueUpdateAfterHandlingErr(err) {
		return
	}

	containers.sortByStateIconThenName()
	widget.Containers = containers
}

func (widget *dockerContainersWidget) Render() template.HTML {
	return widget.renderTemplate(widget, dockerContainersWidgetTemplate)
}

const (
	dockerContainerLabelHide        = "glance.hide"
	dockerContainerLabelName        = "glance.name"
	dockerContainerLabelURL         = "glance.url"
	dockerContainerLabelDescription = "glance.description"
	dockerContainerLabelSameTab     = "glance.same-tab"
	dockerContainerLabelIcon        = "glance.icon"
	dockerContainerLabelID          = "glance.id"
	dockerContainerLabelParent      = "glance.parent"
	dockerContainerLabelCategory    = "glance.category"
)

const (
	dockerContainerStateIconOK     = "ok"
	dockerContainerStateIconPaused = "paused"
	dockerContainerStateIconWarn   = "warn"
	dockerContainerStateIconOther  = "other"
)

var dockerContainerStateIconPriorities = map[string]int{
	dockerContainerStateIconWarn:   0,
	dockerContainerStateIconOther:  1,
	dockerContainerStateIconPaused: 2,
	dockerContainerStateIconOK:     3,
}

type dockerContainerJsonResponse struct {
	ID     string                `json:"Id"`
	Names  []string              `json:"Names"`
	Image  string                `json:"Image"`
	State  string                `json:"State"`
	Status string                `json:"Status"`
	Labels dockerContainerLabels `json:"Labels"`
}

type dockerContainerLabels map[string]string

func (l *dockerContainerLabels) getOrDefault(label, def string) string {
	if l == nil {
		return def
	}

	v, ok := (*l)[label]
	if !ok {
		return def
	}

	if v == "" {
		return def
	}

	return v
}

type dockerContainer struct {
	Name        string
	URL         string
	SameTab     bool
	Image       string
	State       string
	StateText   string
	StateIcon   string
	Description string
	Icon        customIconField
	Children    dockerContainerList

	rawName       string
	CPUPercent    float64
	MemoryUsedMB  uint64
	MemoryLimitMB uint64
	MemoryPercent uint8
	HasStats      bool
}

type dockerContainerList []dockerContainer

func (containers dockerContainerList) sortByStateIconThenName() {
	p := &dockerContainerStateIconPriorities

	sort.SliceStable(containers, func(a, b int) bool {
		if containers[a].StateIcon != containers[b].StateIcon {
			return (*p)[containers[a].StateIcon] < (*p)[containers[b].StateIcon]
		}

		return strings.ToLower(containers[a].Name) < strings.ToLower(containers[b].Name)
	})
}

func dockerContainerStateToStateIcon(container *dockerContainerJsonResponse) string {
	if strings.Contains(strings.ToLower(container.Status), "(unhealthy)") {
		return dockerContainerStateIconWarn
	}

	switch strings.ToLower(container.State) {
	case "running":
		return dockerContainerStateIconOK
	case "paused":
		return dockerContainerStateIconPaused
	case "exited", "dead":
		return dockerContainerStateIconWarn
	default:
		return dockerContainerStateIconOther
	}
}

func fetchDockerContainers(
	socketPath string,
	hideByDefault bool,
	category string,
	runningOnly bool,
	formatNames bool,
	labelOverrides map[string]map[string]string,
) (dockerContainerList, error) {
	containers, err := fetchDockerContainersFromSource(socketPath, category, runningOnly, labelOverrides)
	if err != nil {
		return nil, fmt.Errorf("fetching containers: %w", err)
	}

	containers, children := groupDockerContainerChildren(containers, hideByDefault)
	dockerContainers := make(dockerContainerList, 0, len(containers))

	for i := range containers {
		container := &containers[i]

		dc := dockerContainer{
			rawName:     strings.TrimLeft(itemAtIndexOrDefault(container.Names, 0, ""), "/"),
			Name:        deriveDockerContainerName(container, formatNames),
			URL:         container.Labels.getOrDefault(dockerContainerLabelURL, ""),
			Description: container.Labels.getOrDefault(dockerContainerLabelDescription, ""),
			SameTab:     stringToBool(container.Labels.getOrDefault(dockerContainerLabelSameTab, "false")),
			Image:       container.Image,
			State:       strings.ToLower(container.State),
			StateText:   strings.ToLower(container.Status),
			Icon:        newCustomIconField(container.Labels.getOrDefault(dockerContainerLabelIcon, "si:docker")),
		}

		if idValue := container.Labels.getOrDefault(dockerContainerLabelID, ""); idValue != "" {
			if children, ok := children[idValue]; ok {
				for i := range children {
					child := &children[i]
					dc.Children = append(dc.Children, dockerContainer{
						Name:      deriveDockerContainerName(child, formatNames),
						StateText: child.Status,
						StateIcon: dockerContainerStateToStateIcon(child),
					})
				}
			}
		}

		dc.Children.sortByStateIconThenName()

		stateIconSupersededByChild := false
		for i := range dc.Children {
			if dc.Children[i].StateIcon == dockerContainerStateIconWarn {
				dc.StateIcon = dockerContainerStateIconWarn
				stateIconSupersededByChild = true
				break
			}
		}
		if !stateIconSupersededByChild {
			dc.StateIcon = dockerContainerStateToStateIcon(container)
		}

		dockerContainers = append(dockerContainers, dc)
	}

	attachDockerContainerStats(socketPath, containers, dockerContainers)

	return dockerContainers, nil
}

// attachDockerContainerStats fills in CPU and memory usage for the containers
// that made it into the rendered list. The stats endpoint blocks for about one
// sampling interval per call, so the requests run concurrently and a failure
// only leaves that container without stats rather than failing the widget.
func attachDockerContainerStats(
	source string,
	sources []dockerContainerJsonResponse,
	targets dockerContainerList,
) {
	if len(targets) == 0 {
		return
	}

	byName := make(map[string]string, len(sources))
	for i := range sources {
		c := &sources[i]
		if len(c.Names) == 0 {
			continue
		}
		byName[strings.TrimLeft(c.Names[0], "/")] = c.ID
	}

	client, scheme, hostname, err := newDockerClient(source)
	if err != nil {
		return
	}

	type result struct {
		index int
		stats *dockerContainerStatsResponse
	}

	results := make(chan result, len(targets))
	var wg sync.WaitGroup

	for i := range targets {
		id := byName[strings.TrimLeft(targets[i].rawName, "/")]
		if id == "" {
			continue
		}

		wg.Add(1)
		go func(index int, containerID string) {
			defer wg.Done()
			stats, err := fetchDockerContainerStats(client, scheme, hostname, containerID)
			if err != nil {
				return
			}
			results <- result{index: index, stats: stats}
		}(i, id)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		if r.stats == nil {
			continue
		}

		dc := &targets[r.index]
		dc.HasStats = true
		dc.CPUPercent = r.stats.cpuPercent()
		dc.MemoryUsedMB = r.stats.memoryUsageBytes() / 1024 / 1024
		dc.MemoryLimitMB = r.stats.MemoryStats.Limit / 1024 / 1024
		if dc.MemoryLimitMB > 0 {
			dc.MemoryPercent = uint8(min(100, dc.MemoryUsedMB*100/dc.MemoryLimitMB))
		}
	}
}

func deriveDockerContainerName(container *dockerContainerJsonResponse, formatNames bool) string {
	if v := container.Labels.getOrDefault(dockerContainerLabelName, ""); v != "" {
		return v
	}

	if len(container.Names) == 0 || container.Names[0] == "" {
		return "n/a"
	}

	name := strings.TrimLeft(container.Names[0], "/")

	if formatNames {
		name = strings.ReplaceAll(name, "_", " ")
		name = strings.ReplaceAll(name, "-", " ")

		words := strings.Split(name, " ")
		for i := range words {
			if len(words[i]) > 0 {
				words[i] = strings.ToUpper(words[i][:1]) + words[i][1:]
			}
		}
		name = strings.Join(words, " ")
	}

	return name
}

func groupDockerContainerChildren(
	containers []dockerContainerJsonResponse,
	hideByDefault bool,
) (
	[]dockerContainerJsonResponse,
	map[string][]dockerContainerJsonResponse,
) {
	parents := make([]dockerContainerJsonResponse, 0, len(containers))
	children := make(map[string][]dockerContainerJsonResponse)

	for i := range containers {
		container := &containers[i]

		if isDockerContainerHidden(container, hideByDefault) {
			continue
		}

		isParent := container.Labels.getOrDefault(dockerContainerLabelID, "") != ""
		parent := container.Labels.getOrDefault(dockerContainerLabelParent, "")

		if !isParent && parent != "" {
			children[parent] = append(children[parent], *container)
		} else {
			parents = append(parents, *container)
		}
	}

	return parents, children
}

func isDockerContainerHidden(container *dockerContainerJsonResponse, hideByDefault bool) bool {
	if v := container.Labels.getOrDefault(dockerContainerLabelHide, ""); v != "" {
		return stringToBool(v)
	}

	return hideByDefault
}

// newDockerClient builds an HTTP client and the scheme/host pair needed to
// talk to the Docker API, which may be a unix socket or a TCP endpoint.
func newDockerClient(source string) (*http.Client, string, string, error) {
	if strings.HasPrefix(source, "tcp://") || strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		parsed, err := url.Parse(source)
		if err != nil {
			return nil, "", "", fmt.Errorf("parsing URL: %w", err)
		}

		scheme := parsed.Scheme
		if scheme == "tcp" {
			scheme = "http"
		}

		port := parsed.Port()
		if port == "" {
			if scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}

		return &http.Client{}, scheme, parsed.Hostname() + ":" + port, nil
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", source)
			},
		},
	}

	return client, "http", "docker", nil
}

func fetchDockerContainersFromSource(
	source string,
	category string,
	runningOnly bool,
	labelOverrides map[string]map[string]string,
) ([]dockerContainerJsonResponse, error) {
	client, scheme, hostname, err := newDockerClient(source)
	if err != nil {
		return nil, err
	}

	fetchAll := ternary(runningOnly, "false", "true")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, "GET", scheme+"://"+hostname+"/containers/json?all="+fetchAll, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("sending request to socket: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 response status: %s", response.Status)
	}

	var containers []dockerContainerJsonResponse
	if err := json.NewDecoder(response.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	for i := range containers {
		container := &containers[i]
		name := strings.TrimLeft(itemAtIndexOrDefault(container.Names, 0, ""), "/")

		if name == "" {
			continue
		}

		overrides, ok := labelOverrides[name]
		if !ok {
			continue
		}

		if container.Labels == nil {
			container.Labels = make(dockerContainerLabels)
		}

		for label, value := range overrides {
			container.Labels["glance."+label] = value
		}
	}

	// We have to filter here instead of using the `filters` parameter of Docker's API
	// because the user may define a category override within their config
	if category != "" {
		filtered := make([]dockerContainerJsonResponse, 0, len(containers))

		for i := range containers {
			container := &containers[i]

			if container.Labels.getOrDefault(dockerContainerLabelCategory, "") == category {
				filtered = append(filtered, *container)
			}
		}

		containers = filtered
	}

	return containers, nil
}

// dockerContainerStatsResponse holds the subset of Docker's /containers/{id}/stats
// payload needed to compute CPU and memory usage.
type dockerContainerStatsResponse struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     uint32 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64            `json:"usage"`
		Limit uint64            `json:"limit"`
		Stats map[string]uint64 `json:"stats"`
	} `json:"memory_stats"`
}

// fetchDockerContainerStats queries a single container's stats endpoint.
// The endpoint blocks for roughly one sampling interval when stream is false,
// so callers must fetch these concurrently and allow a generous timeout.
func fetchDockerContainerStats(client *http.Client, scheme, hostname, id string) (*dockerContainerStatsResponse, error) {
	request, err := http.NewRequest(
		"GET",
		scheme+"://"+hostname+"/containers/"+url.PathEscape(id)+"/stats?stream=false",
		nil,
	)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 response status: %s", response.Status)
	}

	var stats dockerContainerStatsResponse
	if err := json.NewDecoder(response.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// cpuPercent mirrors the calculation the Docker CLI uses: the change in the
// container's CPU time over the change in system CPU time, scaled by core count.
func (s *dockerContainerStatsResponse) cpuPercent() float64 {
	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage) - float64(s.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(s.CPUStats.SystemCPUUsage) - float64(s.PreCPUStats.SystemCPUUsage)

	if cpuDelta <= 0 || systemDelta <= 0 {
		return 0
	}

	cores := float64(s.CPUStats.OnlineCPUs)
	if cores == 0 {
		cores = 1
	}

	return (cpuDelta / systemDelta) * cores * 100
}

// memoryUsageBytes returns working-set memory, excluding page cache, which is
// what the Docker CLI reports as container memory usage.
func (s *dockerContainerStatsResponse) memoryUsageBytes() uint64 {
	usage := s.MemoryStats.Usage

	cache := s.MemoryStats.Stats["inactive_file"]
	if cache == 0 {
		cache = s.MemoryStats.Stats["cache"]
	}

	if cache >= usage {
		return 0
	}

	return usage - cache
}

// agentContainerStats mirrors the payload of the glance-agent /api/containers
// endpoint, which reports state plus resource usage for running containers.
type agentContainerStats struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Image         string  `json:"image"`
	State         string  `json:"state"`
	Status        string  `json:"status"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsedMB  uint64  `json:"memory_used_mb"`
	MemoryLimitMB uint64  `json:"memory_limit_mb"`
	MemoryPercent uint8   `json:"memory_percent"`
}

// fetchContainersFromAgent reads container state and usage from a remote
// glance-agent. The agent exposes a read-only view, so no Docker socket has to
// be reachable from this host.
func fetchContainersFromAgent(agentURL string, labelOverrides map[string]map[string]string) (dockerContainerList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

	containers := make(dockerContainerList, 0, len(stats))

	for i := range stats {
		entry := &stats[i]

		overrides := labelOverrides[entry.Name]
		labels := dockerContainerLabels{}
		for label, value := range overrides {
			labels["glance."+label] = value
		}

		if stringToBool(labels.getOrDefault(dockerContainerLabelHide, "false")) {
			continue
		}

		name := labels.getOrDefault(dockerContainerLabelName, "")
		if name == "" {
			name = entry.Name
			name = strings.ReplaceAll(name, "_", " ")
			name = strings.ReplaceAll(name, "-", " ")
			words := strings.Split(name, " ")
			for j := range words {
				if len(words[j]) > 0 {
					words[j] = strings.ToUpper(words[j][:1]) + words[j][1:]
				}
			}
			name = strings.Join(words, " ")
		}

		dc := dockerContainer{
			Name:          name,
			URL:           labels.getOrDefault(dockerContainerLabelURL, ""),
			Description:   labels.getOrDefault(dockerContainerLabelDescription, ""),
			SameTab:       stringToBool(labels.getOrDefault(dockerContainerLabelSameTab, "false")),
			Image:         entry.Image,
			State:         entry.State,
			StateText:     entry.Status,
			Icon:          newCustomIconField(labels.getOrDefault(dockerContainerLabelIcon, "si:docker")),
			CPUPercent:    entry.CPUPercent,
			MemoryUsedMB:  entry.MemoryUsedMB,
			MemoryLimitMB: entry.MemoryLimitMB,
			MemoryPercent: entry.MemoryPercent,
			HasStats:      entry.MemoryUsedMB > 0,
		}

		if strings.Contains(strings.ToLower(entry.Status), "(unhealthy)") {
			dc.StateIcon = dockerContainerStateIconWarn
		} else {
			switch entry.State {
			case "running":
				dc.StateIcon = dockerContainerStateIconOK
			case "paused":
				dc.StateIcon = dockerContainerStateIconPaused
			case "exited", "dead":
				dc.StateIcon = dockerContainerStateIconWarn
			default:
				dc.StateIcon = dockerContainerStateIconOther
			}
		}

		containers = append(containers, dc)
	}

	return containers, nil
}
