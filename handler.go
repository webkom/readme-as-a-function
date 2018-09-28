package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	graphql "github.com/graph-gophers/graphql-go"
	"time"
)

const schema = `
schema {
	query: Query
}
type Query {
	# Get a list of readmes
	readmeUtgaver(
		# Filter by year
		year: Int
		# filter by issue number, 1 to 6
		utgave: Int
		# Get the first _n_ issues/utgaver
		first: Int
	): [ReadmeUtgave!]!
	# Get the latest readme
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

type resolver struct {
	readmes []ReadmeUtgave
	err     error
}

// ReadmeUtgave is a cool struct
type ReadmeUtgave struct {
	Title  string
	Image  string
	Pdf    string
	Year   int32
	Utgave int32
}

// TITLE returns the title of the readme
func (r ReadmeUtgave) TITLE() string {
	return r.Title
}

// PDF returns the complete url of the pdf
func (r ReadmeUtgave) PDF() string {
	return url + r.Pdf
}

// IMAGE returns the complete url of the image
func (r ReadmeUtgave) IMAGE() string {
	return url + r.Image
}

// YEAR returns the year
func (r ReadmeUtgave) YEAR() int32 {
	return r.Year
}

// UTGAVE returns the "utgave" nr
func (r ReadmeUtgave) UTGAVE() int32 {
	return r.Utgave
}

func (r *resolver) LatestReadme() (*ReadmeUtgave, error) {
	if r.err != nil {
		return nil, r.err
	}
	if len(r.readmes) == 0 {
		return nil, nil
	}
	return &r.readmes[0], nil
}

func (r *resolver) ReadmeUtgaver(input *struct {
	Year   *int32
	Utgave *int32
	First  *int32
}) ([]ReadmeUtgave, error) {
	if r.err != nil {
		return nil, r.err
	}
	if input == nil {
		return r.readmes, nil
	}
	filteredReadmes := make([]ReadmeUtgave, 0)
	for _, r := range r.readmes {
		if input.Year != nil && *input.Year != r.Year {
			continue
		}
		if input.Utgave != nil && *input.Utgave != r.Utgave {
			continue
		}
		filteredReadmes = append(filteredReadmes, r)
		if input.First != nil && len(filteredReadmes) == int(*input.First) {
			break
		}
	}

	return filteredReadmes, nil
}
func sortReadmes(utgaver *[]ReadmeUtgave) {
	sort.Slice(*utgaver, func(i, j int) bool {
		if (*utgaver)[i].Year == (*utgaver)[j].Year {
			return (*utgaver)[i].Utgave > (*utgaver)[j].Utgave
		}
		return (*utgaver)[i].Year > (*utgaver)[j].Year
	})
}

func getSortedReadmes(data io.Reader) []ReadmeUtgave {
	doc, err := goquery.NewDocumentFromReader(data)
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

	sortReadmes(&utgaver)

	return utgaver
}

var params struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

var graphiql = `
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
				return fetch("/", {
					method: "POST",
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

const cancelTimeout = 8 * time.Second

func fetchReadmes(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return res.Body, err

}

// Handle a serverless request
func Handle(req []byte) string {
	var params struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}
	if err := json.Unmarshal(req, &params); err != nil {
		return graphiql
	}
	ctx, cancel := context.WithTimeout(context.Background(), cancelTimeout)
	defer cancel()

	var readmes []ReadmeUtgave
	dataReader, err := fetchReadmes(ctx, url)
	defer dataReader.Close()
	if err == nil {
		readmes = getSortedReadmes(dataReader)
	}
	r := resolver{
		err:     err,
		readmes: readmes,
	}

	s := graphql.MustParseSchema(schema, &r)
	response := s.Exec(ctx, params.Query, params.OperationName, params.Variables)
	responseJSON, _ := json.Marshal(response)

	return string(responseJSON)

}

//func main() {
//	input := `
//	{"query":"{\n  readmeUtgaver{\n    title\n    year\n    image\n    pdf\n  }\n}","variables":null,"operationName":null}`
//	fmt.Println(Handle([]byte(input)))
//}
