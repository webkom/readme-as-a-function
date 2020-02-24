package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"regexp"
	"log"

	graphql "github.com/graph-gophers/graphql-go"
	"time"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
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

type resolver struct {
	client storage.Client
	ctx context.Context
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
	return strings.ReplaceAll(r.Pdf, "%", "%%")
}

// IMAGE returns the complete url of the image
func (r ReadmeUtgave) IMAGE() string {
	return strings.ReplaceAll(r.Image, "%", "%%")
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
	readmes, err := getReadmes(r.ctx, r.client, "")
	sortReadmes(&readmes)
	if len(readmes) == 0 {
		return nil, nil
	}
	return &readmes[0], err
}

// ReadmeUtgaveFilter is the filter
type ReadmeUtgaveFilter struct {
	Year   *int32
	Utgave *int32
	First  *int32
}

func getReadmes(ctx context.Context, client storage.Client, query string) ([]ReadmeUtgave, error) {
	pdfQuery := &storage.Query{Prefix: "pdf/" + query}
	imageQuery := &storage.Query{Prefix: "images/" + query}

	re := regexp.MustCompile(`(?P<Year>\d{4})-(?P<Utgave>\d{2})`)

	bkt := client.Bucket("readme-arkiv.appspot.com")
	pdfIt := bkt.Objects(ctx, pdfQuery)
	imageIt := bkt.Objects(ctx, imageQuery)
	var err error
	var readmes  []ReadmeUtgave
	for {
		pdfAttrs, err := pdfIt.Next()
		if err == iterator.Done {
			break
		}
		imageAttrs, err := imageIt.Next()

		if err == iterator.Done {
			break
		}

		// Object may be a directory
		match := re.FindAllString(pdfAttrs.Name, -1)
		if len(match) == 0 {
			continue
		}

		matches := re.FindStringSubmatch(pdfAttrs.Name)
		utgave, _ := strconv.ParseInt(matches[2], 10, 32)
		year, _ := strconv.ParseInt(matches[1], 10, 32)
		title := re.FindAllString(pdfAttrs.Name, 1)[0]

		r := ReadmeUtgave{
			Title: title,
			Image: imageAttrs.MediaLink,
			Pdf: pdfAttrs.MediaLink,
			Year: int32(year),
			Utgave: int32(utgave),
		}
		readmes = append(readmes, r)
	}
	if err == iterator.Done {
		err = nil
	}
	return readmes, err
}

func (r *resolver) ReadmeUtgaver(filter *ReadmeUtgaveFilter) ([]ReadmeUtgave, error) {
	if filter == nil {
		return getReadmes(r.ctx, r.client, "")
	}
	var query string
	var readmes []ReadmeUtgave
	var err error
	if filter.Year != nil {
		query = fmt.Sprintf("%d/", *filter.Year)
		readmes, err = getReadmes(r.ctx, r.client, query)
	} else {
		readmes, err = getReadmes(r.ctx, r.client, "")
	}
	if filter.Utgave != nil {
		readmes = filterReadmes(readmes, filter)
	}
	sortReadmes(&readmes)
	if filter.First != nil {
		return readmes[:*filter.First], err
	}

	return readmes, err
}

func sortReadmes(utgaver *[]ReadmeUtgave) {
	sort.Slice(*utgaver, func(i, j int) bool {
		if (*utgaver)[i].Year == (*utgaver)[j].Year {
			return (*utgaver)[i].Utgave > (*utgaver)[j].Utgave
		}
		return (*utgaver)[i].Year > (*utgaver)[j].Year
	})
}

func filterReadmes(utgaver []ReadmeUtgave, filter *ReadmeUtgaveFilter) []ReadmeUtgave {
	var res []ReadmeUtgave
	for _, utgave := range utgaver {
		if filter.Utgave != nil && *filter.Utgave != utgave.Utgave {
			continue
		}
		if filter.Year != nil && *filter.Year != utgave.Year {
			continue
		}

		res = append(res, utgave)
	}
	return res
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
		<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.12.0/graphiql.css" />
		<script src="https://cdnjs.cloudflare.com/ajax/libs/fetch/1.1.0/fetch.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/react/15.5.4/react.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/react/15.5.4/react-dom.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.12.0/graphiql.js"></script>
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


// Handle a serverless request
func Handle(req []byte) string {
	var params struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}

	// If the request is empty / GET
	if len(req) == 0 {
		return graphiql
	}

	var err error
	err = json.Unmarshal(req, &params)
	if err != nil {
		return renderError(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cancelTimeout)

	defer cancel()

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}

	r := resolver{
		*client,
		ctx,
	}

	s := graphql.MustParseSchema(schema, &r)
	response := s.Exec(ctx, params.Query, params.OperationName, params.Variables)
	responseJSON, _ := json.Marshal(response)

	return string(responseJSON)

}

func renderError(err error) string {
	// TODO fix printng
	// log.Printf("Unexpected error occurred %e\n", err)
	return fmt.Sprintf(`{"errors":[{"message":"%s"}]}`, err.Error())
}
