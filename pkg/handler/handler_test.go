package handler

import (
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
)

var offline = flag.Bool("offline", false, "Offline test - does not fetch remote data")

type graphqlResponse struct {
	Data struct {
		ReadmeUtgaver []ReadmeUtgave `json:"readmeUtgaver"`
		LatestReadme  *ReadmeUtgave  `json:"latestReadme"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// newEditionsServer serves body with the given status as a fake editions API.
func newEditionsServer(t *testing.T, status int, body string) *resolver {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request to the editions API, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return &resolver{client: server.Client(), editionsURL: server.URL}
}

func fixtureResolver(t *testing.T) *resolver {
	t.Helper()
	body, err := os.ReadFile("testdata/editions.json")
	if err != nil {
		t.Fatalf("Unable to read fixture: %s", err)
	}
	return newEditionsServer(t, http.StatusOK, string(body))
}

func query(t *testing.T, r *resolver, q string) graphqlResponse {
	t.Helper()
	req, err := json.Marshal(map[string]string{"query": q})
	if err != nil {
		t.Fatal(err)
	}
	raw := handle(req, r)
	var res graphqlResponse
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		t.Fatalf("Unable to parse response %s: %s", raw, err)
	}
	return res
}

func titles(readmes []ReadmeUtgave) []string {
	res := []string{}
	for _, r := range readmes {
		res = append(res, r.Title)
	}
	return res
}

func TestFieldMapping(t *testing.T) {
	res := query(t, fixtureResolver(t), `{readmeUtgaver(first: 1){title image pdf year utgave}}`)
	if len(res.Errors) != 0 {
		t.Fatalf("Expected no errors, got %+v", res.Errors)
	}
	expected := []ReadmeUtgave{{
		Title:  "readme utgave nr. 2 2026",
		Image:  "https://firebasestorage.googleapis.com/v0/b/readme-arkiv.appspot.com/o/images%2F2026%2F2026-02.jpg?alt=media",
		Pdf:    "https://readme.abakus.no/edition/2026-02",
		Year:   2026,
		Utgave: 2,
	}}
	if !reflect.DeepEqual(res.Data.ReadmeUtgaver, expected) {
		t.Errorf("Expected %+v, got %+v", expected, res.Data.ReadmeUtgaver)
	}
}

func TestReadmeUtgaver(t *testing.T) {
	testCases := []struct {
		name  string
		query string
		out   []string
	}{
		{
			name:  "All, newest first, skipping invalid editions",
			query: `{readmeUtgaver{title}}`,
			out: []string{
				"readme utgave nr. 2 2026",
				"readme utgave nr. 1 2026",
				"readme utgave nr. 6 2025",
				"readme utgave nr. 1 2025",
				"readme utgave nr. 1 1999",
			},
		},
		{
			name:  "First 2",
			query: `{readmeUtgaver(first: 2){title}}`,
			out:   []string{"readme utgave nr. 2 2026", "readme utgave nr. 1 2026"},
		},
		{
			name:  "First 0",
			query: `{readmeUtgaver(first: 0){title}}`,
			out:   []string{},
		},
		{
			name:  "First more than available",
			query: `{readmeUtgaver(first: 100){title}}`,
			out: []string{
				"readme utgave nr. 2 2026",
				"readme utgave nr. 1 2026",
				"readme utgave nr. 6 2025",
				"readme utgave nr. 1 2025",
				"readme utgave nr. 1 1999",
			},
		},
		{
			name:  "All from 2025",
			query: `{readmeUtgaver(year: 2025){title}}`,
			out:   []string{"readme utgave nr. 6 2025", "readme utgave nr. 1 2025"},
		},
		{
			name:  "Utgave 1 from every year",
			query: `{readmeUtgaver(utgave: 1){title}}`,
			out:   []string{"readme utgave nr. 1 2026", "readme utgave nr. 1 2025", "readme utgave nr. 1 1999"},
		},
		{
			name:  "First utgave 1",
			query: `{readmeUtgaver(utgave: 1, first: 1){title}}`,
			out:   []string{"readme utgave nr. 1 2026"},
		},
		{
			name:  "Utgave 6 from 2025",
			query: `{readmeUtgaver(year: 2025, utgave: 6){title}}`,
			out:   []string{"readme utgave nr. 6 2025"},
		},
		{
			name:  "Year without editions",
			query: `{readmeUtgaver(year: 2000){title}}`,
			out:   []string{},
		},
	}
	r := fixtureResolver(t)
	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			res := query(t, r, c.query)
			if len(res.Errors) != 0 {
				t.Fatalf("Expected no errors, got %+v", res.Errors)
			}
			if got := titles(res.Data.ReadmeUtgaver); !reflect.DeepEqual(got, c.out) {
				t.Errorf("Expected %+v, got %+v", c.out, got)
			}
		})
	}
}

func TestReadmeUtgaverNegativeFirst(t *testing.T) {
	res := query(t, fixtureResolver(t), `{readmeUtgaver(first: -1){title}}`)
	if len(res.Errors) != 1 || res.Errors[0].Message != "first must be a non-negative integer" {
		t.Errorf("Expected a single error about first, got %+v", res.Errors)
	}
}

func TestLatestReadme(t *testing.T) {
	res := query(t, fixtureResolver(t), `{latestReadme{title}}`)
	if len(res.Errors) != 0 {
		t.Fatalf("Expected no errors, got %+v", res.Errors)
	}
	if res.Data.LatestReadme == nil || res.Data.LatestReadme.Title != "readme utgave nr. 2 2026" {
		t.Errorf("Expected readme utgave nr. 2 2026 as latest readme, got %+v", res.Data.LatestReadme)
	}
}

func TestLatestReadmeWithoutEditions(t *testing.T) {
	res := query(t, newEditionsServer(t, http.StatusOK, `{"editions":[]}`), `{latestReadme{title}}`)
	if len(res.Errors) != 0 {
		t.Fatalf("Expected no errors, got %+v", res.Errors)
	}
	if res.Data.LatestReadme != nil {
		t.Errorf("Expected no latest readme, got %+v", res.Data.LatestReadme)
	}
}

func TestUpstreamErrors(t *testing.T) {
	testCases := []struct {
		name   string
		status int
		body   string
		err    string
	}{
		{
			name:   "Server error",
			status: http.StatusInternalServerError,
			body:   `{}`,
			err:    "fetching readme editions: unexpected status 500 Internal Server Error",
		},
		{
			name:   "Invalid JSON",
			status: http.StatusOK,
			body:   `<html></html>`,
			err:    "decoding readme editions: invalid character '<' looking for beginning of value",
		},
		{
			name:   "Missing editions",
			status: http.StatusOK,
			body:   `{"items":[]}`,
			err:    "decoding readme editions: response has no editions",
		},
	}
	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			r := newEditionsServer(t, c.status, c.body)
			for _, q := range []string{`{readmeUtgaver{title}}`, `{latestReadme{title}}`} {
				res := query(t, r, q)
				if len(res.Errors) != 1 || res.Errors[0].Message != c.err {
					t.Errorf("%s: expected error %q, got %+v", q, c.err, res.Errors)
				}
				if len(res.Data.ReadmeUtgaver) != 0 || res.Data.LatestReadme != nil {
					t.Errorf("%s: expected no data, got %+v", q, res.Data)
				}
			}
		})
	}
}

func TestUnreachableUpstream(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	r := &resolver{client: server.Client(), editionsURL: server.URL}
	server.Close()

	res := query(t, r, `{readmeUtgaver{title}}`)
	if len(res.Errors) != 1 || !strings.HasPrefix(res.Errors[0].Message, "fetching readme editions: ") {
		t.Errorf("Expected a single fetch error, got %+v", res.Errors)
	}
}

func TestBadInput(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		out  string
	}{
		{
			name: "Empty request body / GET",
			in:   "",
			out:  graphiql,
		},
		{
			name: "Invalid JSON body",
			in:   `{"query": `,
			out:  `{"errors":[{"message":"unexpected end of JSON input"}]}`,
		},
		{
			name: "Invalid syntax inside query",
			in:   `{"query": "{latestReadme{{}}}"}`,
			out:  `{"errors":[{"message":"syntax error: unexpected \"{\", expecting Ident","locations":[{"line":1,"column":15}]}]}`,
		},
		{
			name: "Empty selection set",
			in:   `{"query": "{}"}`,
			out:  `{"errors":[{"message":"syntax error: unexpected \"}\", expecting Ident","locations":[{"line":1,"column":2}]}]}`,
		},
		{
			name: "missing query operation",
			in:   "{}",
			out:  `{"errors":[{"message":"no operations in query document"}]}`,
		},
	}
	r := fixtureResolver(t)
	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			if res := handle([]byte(c.in), r); res != c.out {
				t.Errorf("Expected %s, got %s", c.out, res)
			}
		})
	}
}

// TestLiveEditionsAPI checks that the real editions API still matches what
// this function expects.
func TestLiveEditionsAPI(t *testing.T) {
	if *offline {
		t.Skip("Skipping test against the live editions API")
	}
	res := query(t, &resolver{client: http.DefaultClient, editionsURL: editionsURL}, `{readmeUtgaver{title image pdf year utgave}}`)
	if len(res.Errors) != 0 {
		t.Fatalf("Expected no errors, got %+v", res.Errors)
	}
	// readme has been published since 1999, with up to six editions a year
	if len(res.Data.ReadmeUtgaver) < 100 {
		t.Fatalf("Expected at least 100 readmes, got %d", len(res.Data.ReadmeUtgaver))
	}
	for _, r := range res.Data.ReadmeUtgaver {
		if r.Year < 1999 || r.Utgave < 1 || r.Utgave > 6 {
			t.Errorf("Unexpected year or utgave in %+v", r)
		}
		image, err := url.Parse(r.Image)
		if err != nil || image.Scheme != "https" || image.Host == "" {
			t.Errorf("Expected an absolute https image URL in %+v", r)
		}
		pdf, err := url.Parse(r.Pdf)
		if err != nil || pdf.Scheme != "https" || pdf.Host != "readme.abakus.no" {
			t.Errorf("Expected a readme.abakus.no edition URL in %+v", r)
		}
	}
}
