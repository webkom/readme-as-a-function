package main

import (
	"encoding/json"
	"errors"
	"flag"
	"strings"
	"testing"
)

const in = `
<h2>2017</h2>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2017/2017-01.pdf"><img class="img-responsive" src="bilder/2017/2017-01.jpg" alt="readme utgave nr. 1 2017" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2017/2017-02.pdf"><img class="img-responsive" src="bilder/2017/2017-02.jpg" alt="readme utgave nr. 2 2017" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2017/2017-03.pdf"><img class="img-responsive" src="bilder/2017/2017-03.jpg" alt="readme utgave nr. 3 2017" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2017/2017-04.pdf"><img class="img-responsive" src="bilder/2017/2017-04.jpg" alt="readme utgave nr. 4 2017" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2017/2017-05.pdf"><img class="img-responsive" src="bilder/2017/2017-05.jpg" alt="readme utgave nr. 5 2017" /></a></div>
<div class="col-md-2 col-sm-4 col-xs-6"><a href="utgaver/2016/2016-01.pdf"><img class="img-responsive" src="bilder/2016/2016-01.jpg" alt="readme utgave nr. 1 2016" /></a></div>
</div>
`
const inCount = 6

var offline = flag.Bool("offline", false, "Offline test - does not fetch remote data")

func TestRealParsing(t *testing.T) {
	if offline == nil || *offline {
		t.Skip()
	}
	input := `
	{"query":"{readmeUtgaver{utgave title year image pdf}}","variables":null,"operationName":null}`
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
func TestParsing(t *testing.T) {
	r := strings.NewReader(in)
	readmes := getSortedReadmes(r)
	if len(readmes) != inCount {
		t.Errorf("Expected %d readmes, got %d", inCount, len(readmes))
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

}

func TestBadInput(t *testing.T) {
	if res := Handle([]byte("")); res != graphiql {
		t.Errorf("Expected graphiql on bad input, got %+v\n", res)

	}

}
