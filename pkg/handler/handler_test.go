package handler

import (
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"google.golang.org/api/option"
	"reflect"
	"testing"
)

var offline = flag.Bool("offline", false, "Offline test - does not fetch remote data")

func TestRealData(t *testing.T) {
	if offline == nil || *offline {
		t.Skip()
	}
	input := `{"query":"{readmeUtgaver{utgave title year image pdf}}","variables":null,"operationName":null}`
	res := Handle([]byte(input))
	var result struct {
		Data  map[string]interface{} `json:"data"`
		Error map[string]interface{} `json:"error"`
	}
	if err := json.Unmarshal([]byte(res), &params); err != nil {
		t.Errorf("Error while executing function %e", err)

	}
	if len(result.Error) != 0 {
		t.Errorf("Got the following errors while fetching %+v", result.Error)
	}
}

type StubErrorReader struct{}

var ErrStubReader = errors.New("Wrong")

func (StubErrorReader) Read(p []byte) (n int, err error) {
	return 0, ErrStubReader
}

func TestRegex(t *testing.T) {
	testCases := []struct {
		name    string
		matches []string
	}{
		{
			name:    "2019-03.pdf",
			matches: []string{"2019", "03"},
		},
		{
			name:    "2009-01.jpg",
			matches: []string{"2009", "01"},
		},
		{
			name:    "2009-01.pdf",
			matches: []string{"2009", "01"},
		},
		{
			name:    "2020-01.pdf",
			matches: []string{"2020", "01"},
		},
		{
			name:    "1993-02.pdf",
			matches: []string{"1993", "02"},
		},
		{
			name:    "utgave-1993-02.pdf",
			matches: []string{"1993", "02"},
		},
		{
			name:    "2019-1.pdf",
			matches: nil,
		},
		{
			name:    "02-2019.pdf",
			matches: nil,
		},
		{
			name:    "2019_03.txt",
			matches: nil,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			res := getRegexMatches(c.name)
			if !reflect.DeepEqual(res, c.matches) {
				t.Errorf("Expected %s, got %s\n", c.matches, res)
			}
		})
	}

}

func TestGetReadmes(t *testing.T) {
	if offline == nil || *offline {
		t.Skip()
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithoutAuthentication())
	if err != nil {
		t.Errorf("Error while creating storage client: %+v", err.Error())
	}

	readmes, err := getReadmes(ctx, *client, "2019")

	if err != nil {
		t.Errorf("Expected no errors but got: %s", err.Error())
	}

	if len(readmes) != 6 {
		t.Errorf("Expected exactly 6 readmes, but got %v", len(readmes))
	}
}

func TestReadmeUtgaveResolvers(t *testing.T) {
	r := ReadmeUtgave{
		Title:  "title",
		Year:   2018,
		Utgave: 2,
		Image:  "test.png",
		Pdf:    "2.pdf",
	}
	if res := r.YEAR(); res != r.Year {
		t.Errorf("Expected year %d, got %d", res, r.Year)
	}
	if res := r.TITLE(); res != r.Title {
		t.Errorf("Expected Title %s, got %s", res, r.Title)
	}
	if res := r.UTGAVE(); res != r.Utgave {
		t.Errorf("Expected utgave %d, got %d", res, r.Utgave)
	}
	if res := r.IMAGE(); res != "test.png" {
		t.Errorf("Expected image %s, got %s", res, r.Image)
	}
	if res := r.PDF(); res != "2.pdf" {
		t.Errorf("Expected pdf %s, got %s", res, r.Pdf)
	}
}

func TestLatestReadmeOnline(t *testing.T) {
	if offline == nil || *offline {
		t.Skip()
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithoutAuthentication())
	if err != nil {
		t.Errorf("Error while creating storage client: %+v", err.Error())
	}

	r := resolver{
		client: *client,
		ctx:    ctx,
	}

	first, err := r.LatestReadme()
	if err != nil {
		t.Errorf("Got error %e\n", err)
	}
	if first == nil {
		t.Error("Expected to get a first readme, but got nil.")
	}
}

// LatestReadme is a utility function to test without google cloud apis.
func latestReadme(readmes []ReadmeUtgave) (*ReadmeUtgave, error) {
	sortReadmes(&readmes)
	if len(readmes) == 0 {
		return nil, nil
	}
	return &readmes[0], nil
}

func TestLatestReadme(t *testing.T) {
	rs := []ReadmeUtgave{
		{
			Title: "One",
		},
		{
			Title: "Two",
		},
	}

	first, err := latestReadme(rs)
	if err != nil {
		t.Errorf("Got error %e\n", err)
	}
	if first == nil || first.Title != "One" {
		t.Errorf("Expected %+v as first readme, got %+v\n", rs[0], first)
	}

	first, err = latestReadme([]ReadmeUtgave{})
	if first != nil {
		t.Errorf("Expected no first readme because of empty arr, got %+v\n", first)
	}
	if err != nil {
		t.Errorf("Expected no error %e, got \n", err)
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
			name: "Invalid syntax inside query",
			in:   `{"query": "{latestReadme{{}}}"}`,
			out:  `{"errors":[{"message":"syntax error: unexpected \"{\", expecting Ident","locations":[{"line":1,"column":15}]}]}`,
		},
		{
			name: "valid but empty json, no results",
			in:   `{"query": "{}"}`,
			out:  `{"data":{}}`,
		},
		{
			name: "missing query operation",
			in:   "{}",
			out:  `{"errors":[{"message":"no operations in query document"}]}`,
		},
	}
	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			res := Handle([]byte(c.in))
			if res != c.out {
				t.Errorf("Expected %s, got %s\n", c.out, res)
			}
		})
	}

}

// Util function to convert value to ptr for value
func p(i int32) *int32 {
	return &i
}

// Util function to get title format
func title(year int, utgave int) string {
	return fmt.Sprintf("readme utgave nr. %d %d", utgave, year)
}
func TestReadmeUtgaver(t *testing.T) {
	if offline == nil || *offline {
		t.Skip()
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithoutAuthentication())
	if err != nil {
		t.Errorf("Error while creating storage client: %+v", err.Error())
	}

	res := resolver{
		client: *client,
		ctx:    ctx,
	}

	testCases := []struct {
		name   string
		filter ReadmeUtgaveFilter
		out    []string
		err    error
	}{
		{
			name: "First utgave 3",
			filter: ReadmeUtgaveFilter{
				Utgave: p(3),
				Year:   p(2017),
				First:  p(1),
			},
			out: []string{title(2017, 3)},
		},
		{
			name: "All from 2017",
			filter: ReadmeUtgaveFilter{
				Year: p(2017),
			},
			out: []string{title(2017, 6), title(2017, 5), title(2017, 4), title(2017, 3), title(2017, 2), title(2017, 1)},
		},
		{
			name: "Utgave 3 from 2016",
			filter: ReadmeUtgaveFilter{
				Year:   p(2016),
				Utgave: p(3),
			},
			out: []string{title(2016, 3)},
		},
	}
	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			utgaver, err := res.ReadmeUtgaver(&c.filter)

			var utgaverTitles []string

			for _, utg := range utgaver {
				utgaverTitles = append(utgaverTitles, utg.Title)
			}
			if !reflect.DeepEqual(utgaverTitles, c.out) {
				t.Errorf("Expected %d elements, got %d elements;\n expected %+v, got %+v\n", len(c.out), len(utgaver), c.out, utgaverTitles)
			}
			if err != nil && c.err == nil || err != nil && err.Error() != c.err.Error() {
				t.Errorf("Expected error %e elements, got %e\n", c.err, err)
			}
		})
	}
}
