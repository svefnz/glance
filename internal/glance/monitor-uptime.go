package glance

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"
)

// uptimeBucketCount is the number of buckets the bar always renders. The bucket
// width is derived from the configured window so that the visual density does
// not change when the window changes.
const uptimeBucketCount = 60

// uptimeStore persists per-minute check aggregates.
type uptimeStore struct {
	db *sql.DB
}

const uptimeSchema = `
CREATE TABLE IF NOT EXISTS checks (
	site   TEXT    NOT NULL,
	minute INTEGER NOT NULL,
	ok     INTEGER NOT NULL,
	total  INTEGER NOT NULL,
	avg_ms INTEGER NOT NULL,
	PRIMARY KEY (site, minute)
);
CREATE INDEX IF NOT EXISTS checks_minute ON checks (minute);
`

// openUptimeStore opens the database at path, creating the schema when absent.
// A path that cannot be opened is reported rather than degraded to a memory
// database, so a missing mount does not silently discard history.
func openUptimeStore(path string) (*uptimeStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening uptime database at %q: %w", path, err)
	}

	// SQLite serialises writers; one connection avoids lock contention from
	// the monitor's concurrent update path.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(uptimeSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating uptime schema at %q: %w", path, err)
	}

	return &uptimeStore{db: db}, nil
}

func (s *uptimeStore) Close() error {
	return s.db.Close()
}

// uptimeBucket is one slice of the uptime bar.
type uptimeBucket struct {
	Start    time.Time
	OK       int
	Total    int
	AvgMs    int
	HasData  bool
	Rate     float64
	StateKey string
}

func minuteOf(t time.Time) int64 {
	return t.Unix() / 60
}

// record adds one check to the current minute's row, creating it when absent.
// Repeated checks within a minute accumulate into a single row so the table
// grows with time rather than with check frequency.
func (s *uptimeStore) record(site string, at time.Time, ok bool, responseTime time.Duration) error {
	minute := minuteOf(at)
	okIncr, totalIncr := 0, 1
	if ok {
		okIncr = 1
	}

	ms := int(responseTime.Milliseconds())

	_, err := s.db.Exec(`
		INSERT INTO checks (site, minute, ok, total, avg_ms)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(site, minute) DO UPDATE SET
			avg_ms = (checks.avg_ms * checks.total + excluded.avg_ms) / (checks.total + 1),
			ok     = checks.ok + excluded.ok,
			total  = checks.total + 1
	`, site, minute, okIncr, totalIncr, ms)
	if err != nil {
		return fmt.Errorf("recording check for %q: %w", site, err)
	}

	return nil
}

// bucketWidth is the span of one bar element for a given window.
func bucketWidth(window time.Duration) time.Duration {
	w := window / uptimeBucketCount
	if w < time.Minute {
		w = time.Minute
	}

	return w
}

// uptime returns a fixed number of buckets covering window ending at now.
// Buckets with no recorded checks are reported as having no data rather than
// as healthy, so an empty bar is not mistaken for a perfect record.
func (s *uptimeStore) uptime(site string, window time.Duration, now time.Time) ([]uptimeBucket, error) {
	width := bucketWidth(window)
	// The bar's last bucket contains now, so the first bucket starts at
	// now minus the span of the preceding buckets.
	start := now.Add(-time.Duration(uptimeBucketCount-1) * width).Truncate(width)

	buckets := make([]uptimeBucket, uptimeBucketCount)
	for i := range buckets {
		buckets[i].Start = start.Add(time.Duration(i) * width)
	}

	rows, err := s.db.Query(`
		SELECT minute, ok, total, avg_ms
		FROM checks
		WHERE site = ? AND minute >= ?
		ORDER BY minute
	`, site, minuteOf(start))
	if err != nil {
		return nil, fmt.Errorf("querying uptime for %q: %w", site, err)
	}
	defer rows.Close()

	widthMinutes := int64(width / time.Minute)
	if widthMinutes < 1 {
		widthMinutes = 1
	}

	for rows.Next() {
		var minute int64
		var ok, total, avgMs int
		if err := rows.Scan(&minute, &ok, &total, &avgMs); err != nil {
			return nil, fmt.Errorf("scanning uptime row: %w", err)
		}

		idx := int((minute - minuteOf(start)) / widthMinutes)
		if idx < 0 || idx >= len(buckets) {
			continue
		}

		b := &buckets[idx]
		b.OK += ok
		b.Total += total
		b.AvgMs += avgMs
		b.HasData = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range buckets {
		b := &buckets[i]
		if !b.HasData {
			b.StateKey = "none"
			continue
		}

		b.Rate = float64(b.OK) / float64(b.Total)
		b.StateKey = bucketStateKey(b.Rate)
	}

	return buckets, nil
}

// bucketStateKey maps a success rate onto the bar's colour vocabulary.
func bucketStateKey(rate float64) string {
	switch {
	case rate >= 1:
		return "ok"
	case rate >= 0.9:
		return "degraded"
	default:
		return "failing"
	}
}

// prune deletes rows older than the retention window so the database does not
// grow without bound.
func (s *uptimeStore) prune(keep time.Duration, now time.Time) error {
	cutoff := minuteOf(now.Add(-keep))

	if _, err := s.db.Exec("DELETE FROM checks WHERE minute < ?", cutoff); err != nil {
		return fmt.Errorf("pruning uptime history: %w", err)
	}

	return nil
}

// startPruner runs prune on a schedule in the background. It never blocks the
// update path, so a slow delete cannot stall a widget refresh.
func (s *uptimeStore) startPruner(keep time.Duration, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := s.prune(keep, time.Now()); err != nil {
				slog.Warn("Pruning uptime history", "error", err)
			}
		}
	}()
}
