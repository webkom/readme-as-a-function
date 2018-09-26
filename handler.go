package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	graphql "github.com/graph-gophers/graphql-go"
	"time"
)

type query struct{}

const schema = `
schema {
		query: Query
}
type Query {
	readmeUtgaver(year: Int): [ReadmeUtgave!]!
}
type ReadmeUtgave {
	title: String!
	image: String!
	pdf: String!
	year: Int!
}

`
const url = "https://readme.abakus.no/"

// ReadmeUtgave is a cool struct
type ReadmeUtgave struct {
	Title string
	Image string
	Pdf   string
	Year  int32
}

func (r ReadmeUtgave) TITLE() string {
	return r.Title
}
func (r ReadmeUtgave) PDF() string {
	return url + r.Pdf
}
func (r ReadmeUtgave) IMAGE() string {
	return url + r.Image
}
func (r ReadmeUtgave) YEAR() int32 {
	return r.Year
}

func (q *query) ReadmeUtgaver(ctx context.Context, input *struct{ Year *int32 }) []ReadmeUtgave {
	// Request the HTML page.
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	utgaver := []ReadmeUtgave{}
	// Find the review items
	doc.Find(".col-xs-6").Each(func(i int, s *goquery.Selection) {
		title, _ := s.Find("img").Attr("alt")
		image, _ := s.Find("img").Attr("src")
		pdf, _ := s.Find("a").Attr("href")
		year := int32(0)

		utgave := ReadmeUtgave{
			Year:  year,
			Title: title,
			Pdf:   pdf,
			Image: image,
		}
		utgaver = append(utgaver, utgave)
	})

	return utgaver

}

var params struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// Handle a serverless request
func Handle(req []byte) string {
	var params struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}
	if err := json.Unmarshal(req, &params); err != nil {
		return "RIP"
	}
	s := graphql.MustParseSchema(schema, &query{})
	ctx, _ := context.WithTimeout(context.Background(), 8*time.Second)
	response := s.Exec(ctx, params.Query, params.OperationName, params.Variables)
	responseJSON, _ := json.Marshal(response)

	return string(responseJSON)

}

//func main() {
//	input := `
//	{"query":"{\n  readmeUtgaver{\n    title\n    year\n    image\n    pdf\n  }\n}","variables":null,"operationName":null}`
//	fmt.Println(Handle([]byte(input)))
//}
