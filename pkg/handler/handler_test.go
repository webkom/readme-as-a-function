package handler

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"reflect"
	"testing"
	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
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

func TestLatestReadmeOnline(t *testing.T){
	if offline == nil || *offline {
		t.Skip()
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithoutAuthentication())
	if err != nil {
		t.Errorf("Error while creating storage client: %+v", err.Error())
	}

	r := resolver{
		*client,
		ctx,
	}

	first, err := r.LatestReadme()
	if err != nil {
		t.Errorf("Got error %e\n", err)
	}
	if first == nil {
		t.Error("Expected to get a first readme, but got nil.")
	}
}

// We create a new function that can take test data so that test data is consistent
func LatestReadme(readmes []ReadmeUtgave) (*ReadmeUtgave, error) {
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

	first, err := LatestReadme(rs)
	if err != nil {
		t.Errorf("Got error %e\n", err)
	}
	if first == nil || first.Title != "One" {
		t.Errorf("Expected %+v as first readme, got %+v\n", rs[0], first)
	}

	first, err = LatestReadme([]ReadmeUtgave{})
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
		*client,
		ctx,
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
				Year: p(2017),
				First:  p(1),
			},
			out: []string{"2017-03"},
		},
		{
			name: "All from 2017",
			filter: ReadmeUtgaveFilter{
				Year: p(2017),
			},
			out: []string{"2017-06", "2017-05", "2017-04", "2017-03", "2017-02", "2017-01"},
		},
		{
			name: "Utgave 3 from 2016",
			filter: ReadmeUtgaveFilter{
				Year:   p(2016),
				Utgave: p(3),
			},
			out: []string{"2016-03"},
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
