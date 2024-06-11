package caddy_splunk_hec_log

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	hec "github.com/eventbrite/splunk-hec-go"
	"go.uber.org/zap"
)

const SplunkCollectorHealthPath = "/services/collector/health"

func init() {
	caddy.RegisterModule(SplunkHECLog{})
}

type SplunkHECLog struct {
	// The base URL for the Splunk HEC
	Url string `json:"url,omitempty"`
	// The HEC token used while submitting events
	Token string `json:"token,omitempty"`
	// The duration between flushing any collected events to Splunk
	FlushInterval caddy.Duration `json:"flush_interval,omitempty"`

	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (SplunkHECLog) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "caddy.logging.writers.splunk_hec_log",
		New: func() caddy.Module { return new(SplunkHECLog) },
	}
}

func (l *SplunkHECLog) String() string {
	return "splunk_hec_log"
}

func (l *SplunkHECLog) WriterKey() string {
	return "splunk_hec_log"
}

func (l *SplunkHECLog) OpenWriter() (io.WriteCloser, error) {
	writer := &SplunkHECWriter{
		logger: l.logger,
	}

	go func() {
		writer.Open(l)
	}()

	return writer, nil
}

type SplunkHECWriter struct {
	hostname    string
	logger      *zap.Logger
	client      hec.HEC
	buffer      []*hec.Event
	bufferMutex sync.Mutex
	ticker      *time.Ticker
}

func (l *SplunkHECLog) Provision(ctx caddy.Context) error {
	l.logger = ctx.Logger(l)

	// Set default flush interval when not provided
	if l.FlushInterval == 0 {
		l.FlushInterval = caddy.Duration(10 * time.Second) // Default to 10 seconds
	}

	return nil
}

func (l *SplunkHECLog) Validate() error {
	if l.Url == "" {
		return fmt.Errorf("NO URL SET")
	}

	if l.Token == "" {
		return fmt.Errorf("NO TOKEN SET")
	}

	if l.FlushInterval < 0 {
		return fmt.Errorf("FLUSH_INTERVAL must be greater than 0s")
	}

	return nil
}

func (prom *SplunkHECWriter) Write(p []byte) (n int, err error) {
	f := map[string]interface{}{}
	if err := json.Unmarshal(p, &f); err != nil {
		prom.logger.Error("Unmarshal failed on log", zap.Error((err)))
	}

	event := &hec.Event{
		Host:  &prom.hostname,
		Event: f,
	}

	// buffer event that will be submitted to HEC endpoint
	prom.bufferMutex.Lock()
	prom.buffer = append(prom.buffer, event)
	prom.bufferMutex.Unlock()

	err = prom.client.WriteEvent(event)
	if err != nil {
		prom.logger.Error("Failed to write to Splunk HEC", zap.Error((err)))
	}

	return
}

func (prom *SplunkHECWriter) Close() error {
	// stop ticker from firing further
	prom.ticker.Stop()
	// ensures that any remaining items in the buffer are properly flushed before closing logger
	prom.flushEvents()

	// TODO(derek): this code did not appear to flush when SIGTERM signal was sent, likely not possible to do this.
	// // ensure any un-flushed buffer events are flushed to stderr
	// if len(prom.buffer) > 0 {
	// 	for _, event := range prom.buffer {
	// 		prom.logger.Sugar().Errorln("Event did not reach HEC.",
	// 			"event", event,
	// 		)
	// 	}
	// }

	return nil
}

func (prom *SplunkHECWriter) Open(i *SplunkHECLog) error {
	sugar := prom.logger.Sugar()

	client := hec.NewClient(i.Url, i.Token)
	client.SetCompression("gzip")

	prom.client = client

	// attempt health check against host and validate HEC token
	healthCheckUrl := strings.TrimRight(i.Url, "/") + SplunkCollectorHealthPath
	resp, err := http.Get(healthCheckUrl)
	if err != nil {
		prom.logger.Fatal("Health check failed", zap.Error((err)))
	}
	if resp.StatusCode != http.StatusOK {
		sugar.Fatalf("Health check failed.",
			"http_code", resp.Status,
		)
	}
	defer resp.Body.Close()

	// signal readiness for log module
	sugar.Infof("Health check successful.",
		"url", i.Url,
	)

	// determine hostname
	hostname, err := os.Hostname()
	if err != nil {
		sugar.Fatal("Unable to determine hostname", zap.Error((err)))
	}
	prom.hostname = hostname

	// Start the flush process
	go prom.startFlushTicker(i.FlushInterval)

	return nil
}

// startFlushTicker periodically calls flushEvents() to trigger flushing all events to HEC
func (prom *SplunkHECWriter) startFlushTicker(interval caddy.Duration) {
	ticker := time.NewTicker(time.Duration(interval))
	prom.ticker = ticker

	for range ticker.C {
		prom.flushEvents()
	}
}

// flushEvents writes multiple events via HCE batch mode. Retries are handled by the underlying HEC client rather than this module.
func (prom *SplunkHECWriter) flushEvents() {
	sugar := prom.logger.Sugar()

	// case: no events to flush
	if len(prom.buffer) == 0 {
		sugar.Debugf("Buffer empty, no events flushed.")
		return
	}

	// case: events in buffer ready for flush
	prom.bufferMutex.Lock()
	eventsToFlush := prom.buffer
	prom.buffer = nil
	prom.bufferMutex.Unlock()

	err := prom.client.WriteBatch(eventsToFlush)
	if err != nil {
		sugar.Errorf("Failed to flush events, re-appending events to buffer.",
			"count", len(eventsToFlush),
			"error", zap.Error(err),
		)

		// Re-add events to the buffer
		prom.bufferMutex.Lock()
		prom.buffer = append(eventsToFlush, prom.buffer...)
		prom.bufferMutex.Unlock()
	} else {
		sugar.Debugf("Events flushed.",
			"count", len(eventsToFlush),
		)
	}
}

// Interface guards
var (
	_ caddy.Provisioner     = (*SplunkHECLog)(nil)
	_ caddy.WriterOpener    = (*SplunkHECLog)(nil)
	_ caddyfile.Unmarshaler = (*SplunkHECLog)(nil)
)
