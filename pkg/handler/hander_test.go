package handler

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"reflect"
	"strings"
	"testing"
)

const in = `
<html>
<h2>2017</h2>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2017/2017-01.pdf"><img class="img-responsive" src="bilder/2017/2017-01.jpg" alt="1 2017" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2017/2017-02.pdf"><img class="img-responsive" src="bilder/2017/2017-02.jpg" alt="2 2017" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2017/2017-03.pdf"><img class="img-responsive" src="bilder/2017/2017-03.jpg" alt="3 2017" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2017/2017-04.pdf"><img class="img-responsive" src="bilder/2017/2017-04.jpg" alt="4 2017" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2017/2017-05.pdf"><img class="img-responsive" src="bilder/2017/2017-05.jpg" alt="5 2017" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2017/2017-05.pdf"><img class="img-responsive" src="bilder/2017/2017-05.jpg" alt="6 2017" /></a></div>
<h2>2016</h2>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2016/2016-01.pdf"><img class="img-responsive" src="bilder/2016/2016-01.jpg" alt="1 2016" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2016/2016-02.pdf"><img class="img-responsive" src="bilder/2016/2016-02.jpg" alt="2 2016" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2016/2016-03.pdf"><img class="img-responsive" src="bilder/2016/2016-03.jpg" alt="3 2016" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2016/2016-04.pdf"><img class="img-responsive" src="bilder/2016/2016-04.jpg" alt="4 2016" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2016/2016-05.pdf"><img class="img-responsive" src="bilder/2016/2016-05.jpg" alt="5 2016" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2016/2016-06.pdf"><img class="img-responsive" src="bilder/2016/2016-06.jpg" alt="6 2016" /></a></div>
</div>
</html>
`
const inCount = 12

var offline = flag.Bool("offline", false, "Offline test - does not fetch remote data")

func TestRealParsing(t *testing.T) {
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

var StubErrorReaderError = errors.New("Wrong")

func (StubErrorReader) Read(p []byte) (n int, err error) {
	return 0, StubErrorReaderError
}

func TestParsing(t *testing.T) {
	testCases := []struct {
		name     string
		input    io.Reader
		err      error
		outCount int
	}{
		{
			name:     "Valid html but no results",
			input:    strings.NewReader("<html></html>"),
			outCount: 0,
			err:      parserNoElementsError,
		},
		{
			name:     "Normal parsing",
			input:    strings.NewReader(in),
			outCount: inCount,
		},
		{
			name:     "Bad reader",
			input:    StubErrorReader{},
			err:      StubErrorReaderError,
			outCount: 0,
		},
		{
			name:     "Invalid html",
			input:    strings.NewReader("{\"json\": true}"),
			err:      errors.New("unknown parsing error. No elements found"),
			outCount: 0,
		},
	}
	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			readmes, err := parseReadmes(c.input)
			if len(readmes) != c.outCount {
				t.Errorf("Expected %d readmes, got %d, %+v", c.outCount, len(readmes), readmes)
			}
			if err != nil && c.err == nil || err != nil && err.Error() != c.err.Error() {
				t.Errorf("Expected error %e elements, got %e\n", c.err, err)
			}
		})
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
	if res := r.IMAGE(); res != "https://readme.abakus.no/test.png" {
		t.Errorf("Expected image %s, got %s", res, r.Image)
	}
	if res := r.PDF(); res != "https://readme.abakus.no/2.pdf" {
		t.Errorf("Expected pdf %s, got %s", res, r.Pdf)
	}
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
	r := resolver{readmes: rs, err: nil}
	first, err := r.LatestReadme()
	if err != nil {
		t.Errorf("Got error %e\n", err)
	}
	if first == nil || first.Title != "One" {
		t.Errorf("Expected %+v as first readme, got %+v\n", rs[0], first)
	}

	r = resolver{readmes: rs, err: errors.New("Err")}
	first, err = r.LatestReadme()
	if first != nil {
		t.Errorf("Expected no first readme because of error, got %+v\n", first)
	}
	if err == nil {
		t.Errorf("Expected error %e, got none\n", r.err)

	}

	r = resolver{readmes: []ReadmeUtgave{}, err: nil}
	first, err = r.LatestReadme()
	if first != nil {
		t.Errorf("Expected no first readme beacuse of empty arr, got %+v\n", first)
	}
	if err != nil {
		t.Errorf("Expected no error %e, got \n", r.err)
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
	r := strings.NewReader(in)
	readmes, _ := parseReadmes(r)
	sortReadmes(&readmes)
	res := resolver{
		readmes: readmes,
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
				First:  p(1),
			},
			out: []string{"3 2017"},
		},
		{
			name: "Only utgave 3",
			filter: ReadmeUtgaveFilter{
				Utgave: p(3),
			},
			out: []string{"3 2017", "3 2016"},
		},
		{
			name: "All from 2017",
			filter: ReadmeUtgaveFilter{
				Year: p(2017),
			},
			out: []string{"6 2017", "5 2017", "4 2017", "3 2017", "2 2017", "1 2017"},
		},
		{
			name: "Utgave 3 from 2016",
			filter: ReadmeUtgaveFilter{
				Year:   p(2016),
				Utgave: p(3),
			},
			out: []string{"3 2016"},
		},
		{
			name: "Utgave 3 from all years",
			filter: ReadmeUtgaveFilter{
				Utgave: p(3),
			},
			out: []string{"3 2017", "3 2016"},
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
				t.Errorf("Expected %d elements, got %d elements;\n expected %+v, got %+v\n", len(c.out), len(utgaver), c.out, utgaver)
			}
			if err != nil && c.err == nil || err != nil && err.Error() != c.err.Error() {
				t.Errorf("Expected error %e elements, got %e\n", c.err, err)
			}
		})
	}
}
