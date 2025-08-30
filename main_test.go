package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestAPIEndpoint(t *testing.T) {
	tests := []struct {
		pattern string
		want    int
	}{
		{pattern: "POST /admin/reset", want: 200},
		{pattern: "GET /admin/metrics", want: 200},
		{pattern: "GET /app/", want: 200},
		{pattern: "GET /admin/metrics", want: 200},
		{pattern: "GET /app/assets/", want: 200},
		{pattern: "GET /api/health", want: 200},
		{pattern: "GET /app/", want: 200},
		{pattern: "GET /admin/metrics", want: 200},
	}

	for _, tt := range tests {
		s := strings.Split(tt.pattern, " ")
		m, p := s[0], s[1]
		url := "http://localhost:8080" + p

		t.Run(url, func(t *testing.T) {
			var (
				resp *http.Response
				req  *http.Request
				err  error
			)

			switch m {
			case "GET":
				resp, err = http.Get(url)
			case "POST":
				req, err = http.NewRequest("POST", url, nil)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}
				req.Header.Set("Content-Type", "text/plain; charset=utf-8")
				resp, err = (&http.Client{}).Do(req)
			default:
				t.Fatalf("unsupported method: %s", m)
			}

			if err != nil {
				t.Fatalf("response failed: %v", err)
			}

			defer resp.Body.Close()

			if resp.StatusCode != tt.want {
				t.Errorf("got %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}
