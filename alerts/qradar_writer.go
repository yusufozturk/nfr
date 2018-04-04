package alerts

import (
	"fmt"
	"log/syslog"
	"strings"

	"github.com/alphasoc/nfr/client"
	"github.com/alphasoc/nfr/leef"
	"github.com/alphasoc/nfr/version"
)

const (
	logalert syslog.Priority = 14
	tag                      = "NFR"
)

// QRadarWriter implements Writer interface and write
// api alerts to syslog server.
type QRadarWriter struct {
	w *syslog.Writer
}

// NewQRadarWriter creates new syslog writer.
func NewQRadarWriter(raddr string) (*QRadarWriter, error) {
	w, err := syslog.Dial("tcp", raddr, logalert, tag)
	if err != nil {
		return nil, fmt.Errorf("connect to qradar syslog input failed: %s", err)
	}

	return &QRadarWriter{w: w}, nil
}

// Write writes alert response to the qradar syslog input.
func (w *QRadarWriter) Write(resp *client.AlertsResponse) error {
	if len(resp.Alerts) == 0 {
		return nil
	}

	// send each alert as separate message.
	for _, alert := range resp.Alerts {
		for _, threat := range alert.Threats {
			e := leef.NewEvent()
			e.SetHeader("AlphaSOC", tag, strings.TrimPrefix(version.Version, "v"), threat)

			e.SetSevAttr(resp.Threats[threat].Severity * 2)
			if resp.Threats[threat].Policy {
				e.SetPolicyAttr("1")
			} else {
				e.SetPolicyAttr("0")
			}
			e.SetAttr("flags", strings.Join(alert.Wisdom.Flags, ","))
			e.SetAttr("description", resp.Threats[threat].Title)

			e.SetDevTimeFormatAttr("MMM dd yyyy HH:mm:ss")
			switch alert.EventType {
			case "dns":
				e.SetDevTimeAttr(alert.DNSEvent.Timestamp.Format("Jan 02 2006 15:04:05"))
				e.SetSrcAttr(alert.DNSEvent.SrcIP)
				e.SetAttr("query", alert.DNSEvent.Query)
				e.SetAttr("recordType", alert.DNSEvent.QType)
			case "ip":
				e.SetDevTimeAttr(alert.IPEvent.Timestamp.Format("Jan 02 2006 15:04:05"))
				e.SetProtoAttr(alert.IPEvent.Protocol)
				e.SetSrcAttr(alert.IPEvent.SrcIP)
				e.SetSrcPortAttr(alert.IPEvent.SrcPort)
				e.SetDstAttr(alert.IPEvent.DstIP)
				e.SetDstPortAttr(alert.IPEvent.DstPort)
				e.SetSrcBytesAttr(alert.IPEvent.BytesIn)
				e.SetDstBytesAttr(alert.IPEvent.BytesOut)
			}
			if err := w.w.Alert(e.String()); err != nil {
				return err
			}
		}
	}
	return nil
}

// Close closes a connecion to the syslog server.
func (w *QRadarWriter) Close() error {
	return w.w.Close()
}
