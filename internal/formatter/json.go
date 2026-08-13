package formatter

import (
	"encoding/json"
	"io"

	"github.com/rizkiromadon/envcheck/internal/model"
)

// WriteJSON renders the report as pretty-printed, stable JSON suitable for
// CI/CD pipelines to parse. Field order and shape are defined entirely by
// model.Report's struct tags, so this is a thin, deterministic wrapper.
func WriteJSON(w io.Writer, report model.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
