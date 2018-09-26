package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

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
	readmeUtgaver(year: Int utgave: Int): [ReadmeUtgave!]!
	latestReadme: ReadmeUtgave
}
type ReadmeUtgave {
	title: String!
	image: String!
	pdf: String!
	year: Int!
	utgave: Int!
}

`
const url = "https://readme.abakus.no/"

// ReadmeUtgave is a cool struct
type ReadmeUtgave struct {
	Title  string
	Image  string
	Pdf    string
	Year   int32
	Utgave int32
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
func (r ReadmeUtgave) UTGAVE() int32 {
	return r.Utgave
}
func (q *query) LatestReadme(ctx context.Context) *ReadmeUtgave {
	rs := getSortedReadmes()
	if len(rs) == 0 {
		return nil
	}
	return &rs[0]
}

func (q *query) ReadmeUtgaver(ctx context.Context, input *struct {
	Year   *int32
	Utgave *int32
}) []ReadmeUtgave {
	readmes := getSortedReadmes()
	if input == nil {
		return readmes
	}
	filteredReadmes := readmes[:0]
	for _, r := range readmes {
		if input.Year != nil && *input.Year != r.Year {
			continue
		}
		if input.Utgave != nil && *input.Utgave != r.Utgave {
			continue
		}
		filteredReadmes = append(filteredReadmes, r)
	}

	return filteredReadmes
}

func getSortedReadmes() []ReadmeUtgave {
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
		splitted := strings.Split(title, " ")
		year, _ := strconv.Atoi(splitted[len(splitted)-1])
		utgaveNr, _ := strconv.Atoi(splitted[len(splitted)-2])

		utgave := ReadmeUtgave{
			Year:   int32(year),
			Title:  title,
			Pdf:    pdf,
			Image:  image,
			Utgave: int32(utgaveNr),
		}
		utgaver = append(utgaver, utgave)
	})
	sort.Slice(utgaver, func(i, j int) bool {
		if utgaver[i].Year == utgaver[j].Year {
			return utgaver[i].Utgave > utgaver[j].Utgave
		}
		return utgaver[i].Year > utgaver[j].Year
	})

	return utgaver
}

var params struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

var page = `
<!DOCTYPE html>
<html>
	<head>
		<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.10.2/graphiql.css" />
		<script src="https://cdnjs.cloudflare.com/ajax/libs/fetch/1.1.0/fetch.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/react/15.5.4/react.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/react/15.5.4/react-dom.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.10.2/graphiql.js"></script>
	</head>
	<body style="width: 100%; height: 100%; margin: 0; overflow: hidden;">
		<div id="graphiql" style="height: 100vh;">Loading...</div>
		<script>
			function graphQLFetcher(graphQLParams) {
				return fetch("/graphql", {
					method: "post",
					body: JSON.stringify(graphQLParams),
					credentials: "include",
				}).then(function (response) {
					return response.text();
				}).then(function (responseBody) {
					try {
						return JSON.parse(responseBody);
					} catch (error) {
						return responseBody;
					}
				});
			}

			ReactDOM.render(
				React.createElement(GraphiQL, {fetcher: graphQLFetcher}),
				document.getElementById("graphiql")
			);
		</script>
	</body>
</html>
`

// Handle a serverless request
func Handle(req []byte) string {
	var params struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}
	if err := json.Unmarshal(req, &params); err != nil {
		return page
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
