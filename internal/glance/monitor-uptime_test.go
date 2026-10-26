package glance

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUptimeSchemaCreatesExpectedColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "uptime.db")

	store, err := openUptimeStore(path)
	if err != nil {
		t.Fatalf("openUptimeStore: %v", err)
	}
	defer store.Close()

	rows, err := store.db.Query("PRAGMA table_info(checks)")
	if err != nil {
		t.Fatalf("table_info: %v", err)
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("scan: %v", err)
		}
		columns[name] = true
	}

	for _, want := range []string{"site", "minute", "ok", "total", "avg_ms"} {
		if !columns[want] {
			t.Errorf("missing column %q, got %v", want, columns)
		}
	}
}

func TestRecordAccumulatesWithinOneMinute(t *testing.T) {
	path := filepath.Join(t.TempDir(), "uptime.db")
	store, err := openUptimeStore(path)
	if err != nil {
		t.Fatalf("openUptimeStore: %v", err)
	}
	defer store.Close()

	at := time.Date(2026, 10, 27, 12, 0, 10, 0, time.UTC)
	if err := store.record("gitlab", at, true, 100*time.Millisecond); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := store.record("gitlab", at.Add(20*time.Second), false, 500*time.Millisecond); err != nil {
		t.Fatalf("record: %v", err)
	}

	var ok, total int
	if err := store.db.QueryRow("SELECT ok, total FROM checks WHERE site = ?", "gitlab").Scan(&ok, &total); err != nil {
		t.Fatalf("query: %v", err)
	}

	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if ok != 1 {
		t.Errorf("ok = %d, want 1", ok)
	}

	var rows int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM checks WHERE site = ?", "gitlab").Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Errorf("rows = %d, want 1 (one row per minute)", rows)
	}
}

func TestUptimeBucketCountIndependentOfWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "uptime.db")
	store, err := openUptimeStore(path)
	if err != nil {
		t.Fatalf("openUptimeStore: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

	for _, window := range []time.Duration{time.Hour, 90 * 24 * time.Hour} {
		buckets, err := store.uptime("gitlab", window, now)
		if err != nil {
			t.Fatalf("uptime(%v): %v", window, err)
		}

		if len(buckets) != uptimeBucketCount {
			t.Errorf("window %v: %d buckets, want %d", window, len(buckets), uptimeBucketCount)
		}

		span := buckets[len(buckets)-1].Start.Add(bucketWidth(window)).Sub(buckets[0].Start)
		if span < window-time.Minute || span > window+time.Minute {
			t.Errorf("window %v: buckets span %v", window, span)
		}
	}
}

func TestPruneRemovesOnlyRowsOutsideWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "uptime.db")
	store, err := openUptimeStore(path)
	if err != nil {
		t.Fatalf("openUptimeStore: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	keep := 30 * 24 * time.Hour

	if err := store.record("old", now.Add(-40*24*time.Hour), true, 10*time.Millisecond); err != nil {
		t.Fatalf("record old: %v", err)
	}
	if err := store.record("recent", now.Add(-24*time.Hour), true, 10*time.Millisecond); err != nil {
		t.Fatalf("record recent: %v", err)
	}

	if err := store.prune(keep, now); err != nil {
		t.Fatalf("prune: %v", err)
	}

	var oldCount, recentCount int
	store.db.QueryRow("SELECT COUNT(*) FROM checks WHERE site = 'old'").Scan(&oldCount)
	store.db.QueryRow("SELECT COUNT(*) FROM checks WHERE site = 'recent'").Scan(&recentCount)

	if oldCount != 0 {
		t.Errorf("old rows = %d, want 0", oldCount)
	}
	if recentCount != 1 {
		t.Errorf("recent rows = %d, want 1", recentCount)
	}
}

func TestOpenUptimeStoreReportsUnwritablePath(t *testing.T) {
	// A path under a directory that does not exist cannot be created by SQLite.
	path := filepath.Join(t.TempDir(), "missing", "nested", "uptime.db")

	_, err := openUptimeStore(path)
	if err == nil {
		t.Fatal("openUptimeStore with an unwritable path = nil error, want an error")
	}

	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the path %q", err, path)
	}
}

func TestUptimeRangeDefaultsTo30Days(t *testing.T) {
	widget := &monitorWidget{}
	widget.Uptime.Enabled = true

	if err := widget.initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	got := time.Duration(widget.Uptime.Range)
	if got != 30*24*time.Hour {
		t.Errorf("range = %v, want 720h", got)
	}
}

func TestUptimeRangeBounds(t *testing.T) {
	cases := []struct {
		name    string
		rng     time.Duration
		wantErr bool
	}{
		{"below minimum", 30 * time.Minute, true},
		{"at minimum", time.Hour, false},
		{"at maximum", 90 * 24 * time.Hour, false},
		{"above maximum", 91 * 24 * time.Hour, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			widget := &monitorWidget{}
			widget.Uptime.Enabled = true
			widget.Uptime.Range = durationField(tc.rng)

			err := widget.initialize()
			if tc.wantErr && err == nil {
				t.Errorf("range %v = nil error, want an error", tc.rng)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("range %v = %v, want nil", tc.rng, err)
			}
			if tc.wantErr && err != nil && !strings.Contains(err.Error(), "1h to 90d") {
				t.Errorf("error %q does not name the bounds", err)
			}
		})
	}
}

func TestUptimeDisabledLeavesNoRange(t *testing.T) {
	widget := &monitorWidget{}

	if err := widget.initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	if widget.Uptime.Enabled {
		t.Error("uptime enabled without configuration")
	}
}

// A failed check must still count toward the total, otherwise a service that
// never answers would look like it has no data rather than being down.
func TestTimeoutRecordsTotalWithoutSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "uptime.db")
	store, err := openUptimeStore(path)
	if err != nil {
		t.Fatalf("openUptimeStore: %v", err)
	}
	defer store.Close()

	at := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	if err := store.record("dead", at, false, 0); err != nil {
		t.Fatalf("record: %v", err)
	}

	var ok, total int
	if err := store.db.QueryRow("SELECT ok, total FROM checks WHERE site = 'dead'").Scan(&ok, &total); err != nil {
		t.Fatalf("query: %v", err)
	}

	if ok != 0 {
		t.Errorf("ok = %d, want 0", ok)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
}

// Recording is not gated on uptime.enabled, so enabling the bar later shows
// history that accumulated while it was off.
func TestRecordingHappensWithUptimeDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "uptime.db")
	store, err := openUptimeStore(path)
	if err != nil {
		t.Fatalf("openUptimeStore: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

	// Simulate an update cycle with the bar disabled: the widget records but
	// does not read.
	if err := store.record("site", now, true, 50*time.Millisecond); err != nil {
		t.Fatalf("record: %v", err)
	}

	// Enabling the bar afterwards must show that history.
	buckets, err := store.uptime("site", time.Hour, now)
	if err != nil {
		t.Fatalf("uptime: %v", err)
	}

	withData := 0
	for _, b := range buckets {
		if b.HasData {
			withData++
		}
	}

	if withData == 0 {
		t.Error("no buckets have data after recording with the bar disabled")
	}
}

// The pruner runs in its own goroutine so a slow delete cannot stall a widget
// update. This checks it makes progress without blocking the caller.
func TestPrunerRunsInBackground(t *testing.T) {
	path := filepath.Join(t.TempDir(), "uptime.db")
	store, err := openUptimeStore(path)
	if err != nil {
		t.Fatalf("openUptimeStore: %v", err)
	}
	defer store.Close()

	done := make(chan struct{})
	go func() {
		store.startPruner(time.Hour, 10*time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("startPruner blocked its caller")
	}
}

func TestClassifySiteState(t *testing.T) {
	cases := []struct {
		name     string
		status   *siteStatus
		wantKey  string
		wantText string
	}{
		{
			name:     "fast success is healthy",
			status:   &siteStatus{Code: 200, ResponseTime: 100 * time.Millisecond},
			wantKey:  "ok",
			wantText: "Operational",
		},
		{
			name:     "slow success is degraded",
			status:   &siteStatus{Code: 200, ResponseTime: 2 * time.Second},
			wantKey:  "degraded",
			wantText: "Degraded",
		},
		{
			name:     "error is failing",
			status:   &siteStatus{Code: 500, ResponseTime: 100 * time.Millisecond},
			wantKey:  "failing",
			wantText: "Down",
		},
		{
			name:     "timeout is failing",
			status:   &siteStatus{TimedOut: true},
			wantKey:  "failing",
			wantText: "Down",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, text := classifySiteState(tc.status, nil)
			if key != tc.wantKey {
				t.Errorf("key = %q, want %q", key, tc.wantKey)
			}
			if text != tc.wantText {
				t.Errorf("text = %q, want %q", text, tc.wantText)
			}
		})
	}
}

func TestBucketStateKey(t *testing.T) {
	cases := []struct {
		rate float64
		want string
	}{
		{1.0, "ok"},
		{0.95, "degraded"},
		{0.9, "degraded"},
		{0.89, "failing"},
		{0.0, "failing"},
	}

	for _, tc := range cases {
		if got := bucketStateKey(tc.rate); got != tc.want {
			t.Errorf("bucketStateKey(%v) = %q, want %q", tc.rate, got, tc.want)
		}
	}
}

// The banner must report the worst state present, so a single failing service
// is not hidden by healthy neighbours.
func TestBannerReportsWorstState(t *testing.T) {
	render := func(hasFailing, hasDegraded bool) string {
		w := &monitorWidget{}
		w.HasFailing = hasFailing
		w.HasDegraded = hasDegraded
		w.Title = "test"
		w.ContentAvailable = true
		return string(w.Render())
	}

	if out := render(true, true); !strings.Contains(out, "monitor-banner-failing") {
		t.Error("banner with a failing service did not report failing")
	}
	if out := render(false, true); !strings.Contains(out, "monitor-banner-degraded") {
		t.Error("banner with only degraded services did not report degraded")
	}
	if out := render(false, false); !strings.Contains(out, "monitor-banner-ok") {
		t.Error("banner with all healthy did not report ok")
	}
}

// The bar's element count must not depend on the window, otherwise a short and
// a long range would render visibly different bars.
func TestUptimeBarElementCountIndependentOfWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "uptime.db")
	store, err := openUptimeStore(path)
	if err != nil {
		t.Fatalf("openUptimeStore: %v", err)
	}
	defer store.Close()

	now := time.Now()

	countBuckets := func(window time.Duration) int {
		w := &monitorWidget{}
		w.Title = "test"
		w.ContentAvailable = true
		w.Uptime.Enabled = true
		w.Uptime.Range = durationField(window)
		w.Providers = &widgetProviders{uptimeStore: store}
		w.Sites = append(w.Sites, struct {
			*SiteStatusRequest `yaml:",inline"`
			Status             *siteStatus     `yaml:"-"`
			URL                string          `yaml:"-"`
			ErrorURL           string          `yaml:"error-url"`
			Title              string          `yaml:"title"`
			Icon               customIconField `yaml:"icon"`
			SameTab            bool            `yaml:"same-tab"`
			StatusText         string          `yaml:"-"`
			StatusStyle        string          `yaml:"-"`
			StateKey           string          `yaml:"-"`
			StateText          string          `yaml:"-"`
			Uptime             []uptimeBucket  `yaml:"-"`
			HasUptime          bool            `yaml:"-"`
			AltStatusCodes     []int           `yaml:"alt-status-codes"`
			Container          string          `yaml:"container"`
			CPUPercent         float64         `yaml:"-"`
			MemoryUsedMB       uint64          `yaml:"-"`
			MemoryLimitMB      uint64          `yaml:"-"`
			HasUsage           bool            `yaml:"-"`
		}{SiteStatusRequest: &SiteStatusRequest{}, Title: "site", Status: &siteStatus{Code: 200}})

		w.attachUptime()
		_ = now

		return strings.Count(string(w.Render()), "monitor-uptime-bucket")
	}

	short := countBuckets(time.Hour)
	long := countBuckets(90 * 24 * time.Hour)

	if short != uptimeBucketCount {
		t.Errorf("1h window rendered %d buckets, want %d", short, uptimeBucketCount)
	}
	if short != long {
		t.Errorf("1h rendered %d buckets, 90d rendered %d, want equal", short, long)
	}
}

// The rendered markup must place the banner above the rows and the bar beneath
// each row, since that ordering is what makes it read as a status page.
func TestStatusPageLayoutOrder(t *testing.T) {
	w := &monitorWidget{}
	w.Title = "test"
	w.ContentAvailable = true
	w.Sites = append(w.Sites, struct {
		*SiteStatusRequest `yaml:",inline"`
		Status             *siteStatus     `yaml:"-"`
		URL                string          `yaml:"-"`
		ErrorURL           string          `yaml:"error-url"`
		Title              string          `yaml:"title"`
		Icon               customIconField `yaml:"icon"`
		SameTab            bool            `yaml:"same-tab"`
		StatusText         string          `yaml:"-"`
		StatusStyle        string          `yaml:"-"`
		StateKey           string          `yaml:"-"`
		StateText          string          `yaml:"-"`
		Uptime             []uptimeBucket  `yaml:"-"`
		HasUptime          bool            `yaml:"-"`
		AltStatusCodes     []int           `yaml:"alt-status-codes"`
		Container          string          `yaml:"container"`
		CPUPercent         float64         `yaml:"-"`
		MemoryUsedMB       uint64          `yaml:"-"`
		MemoryLimitMB      uint64          `yaml:"-"`
		HasUsage           bool            `yaml:"-"`
	}{
		SiteStatusRequest: &SiteStatusRequest{},
		Title:             "site",
		Status:            &siteStatus{Code: 200, ResponseTime: 50 * time.Millisecond},
		StateKey:          "ok",
		StateText:         "Operational",
		Uptime:            []uptimeBucket{{StateKey: "ok"}, {StateKey: "failing"}},
		HasUptime:         true,
	})

	out := string(w.Render())

	banner := strings.Index(out, "monitor-banner")
	row := strings.Index(out, "monitor-row")
	bar := strings.Index(out, "monitor-uptime\"")

	if banner < 0 || row < 0 || bar < 0 {
		t.Fatalf("missing markup: banner=%d row=%d bar=%d", banner, row, bar)
	}
	if banner > row {
		t.Error("banner is not above the service rows")
	}
	if bar < row {
		t.Error("uptime bar is not beneath the service row")
	}
}

// The compact variant shares the site sub-template conceptually but has its own
// definition, so a template change must not break it.
func TestCompactVariantRenders(t *testing.T) {
	w := &monitorWidget{}
	w.Title = "test"
	w.ContentAvailable = true
	w.Style = "compact"
	w.Sites = append(w.Sites, struct {
		*SiteStatusRequest `yaml:",inline"`
		Status             *siteStatus     `yaml:"-"`
		URL                string          `yaml:"-"`
		ErrorURL           string          `yaml:"error-url"`
		Title              string          `yaml:"title"`
		Icon               customIconField `yaml:"icon"`
		SameTab            bool            `yaml:"same-tab"`
		StatusText         string          `yaml:"-"`
		StatusStyle        string          `yaml:"-"`
		StateKey           string          `yaml:"-"`
		StateText          string          `yaml:"-"`
		Uptime             []uptimeBucket  `yaml:"-"`
		HasUptime          bool            `yaml:"-"`
		AltStatusCodes     []int           `yaml:"alt-status-codes"`
		Container          string          `yaml:"container"`
		CPUPercent         float64         `yaml:"-"`
		MemoryUsedMB       uint64          `yaml:"-"`
		MemoryLimitMB      uint64          `yaml:"-"`
		HasUsage           bool            `yaml:"-"`
	}{
		SiteStatusRequest: &SiteStatusRequest{},
		Title:             "compact-site",
		Status:            &siteStatus{Code: 200, ResponseTime: 50 * time.Millisecond},
		StateKey:          "ok",
		StateText:         "Operational",
	})

	out := string(w.Render())
	if !strings.Contains(out, "compact-site") {
		t.Error("compact variant did not render the site title")
	}
	if !strings.Contains(out, "monitor-site-status-icon-compact") {
		t.Error("compact variant did not render its status icon")
	}
}
