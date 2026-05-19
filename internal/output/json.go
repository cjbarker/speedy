package output

import (
	"encoding/json"
	"io"

	"github.com/cjbarker/speedy/internal/speedtest"
)

// RenderJSON writes the result as indented JSON.
func RenderJSON(w io.Writer, r *speedtest.Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
