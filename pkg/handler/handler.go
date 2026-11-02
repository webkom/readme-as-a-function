// Package handler serves the readme GraphQL API on top of the public readme
// editions API at https://readme.abakus.no/api/editions.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	graphql "github.com/graph-gophers/graphql-go"
)

const schemaSDL = `
schema {
	query: Query
}
type Query {
	# Get a list of readmes, newest first
	readmeUtgaver(
		# Filter by year
		year: Int
		# Filter by issue number, 1 to 6
		utgave: Int
		# Get the first _n_ issues/utgaver
		first: Int
	): [ReadmeUtgave!]!
	# Get the latest readme
	latestReadme: ReadmeUtgave
}
type ReadmeUtgave {
	title: String!
	# URL of the cover image
	image: String!
	# URL of the edition on readme.abakus.no
	pdf: String!
	year: Int!
	utgave: Int!
}
`

const editionsURL = "https://readme.abakus.no/api/editions"

const requestTimeout = 8 * time.Second

// maxEditionsResponseBytes caps how much of the upstream response is read.
const maxEditionsResponseBytes = 10 << 20

// ReadmeUtgave is a single readme edition.
type ReadmeUtgave struct {
	Title  string
	Image  string
	Pdf    string
	Year   int32
	Utgave int32
}

// edition is an edition as returned by the readme editions API.
type edition struct {
	ID            string `json:"id"`
	EditionNumber int32  `json:"editionNumber"`
	EditionYear   int32  `json:"editionYear"`
	ThumbnailURL  string `json:"thumbnailUrl"`
	EditionURL    string `json:"editionUrl"`
}

type editionsResponse struct {
	Editions []edition `json:"editions"`
}

type resolver struct {
	client      *http.Client
	editionsURL string
}

type readmeUtgaverArgs struct {
	Year   *int32
	Utgave *int32
	First  *int32
}

func (r *resolver) ReadmeUtgaver(ctx context.Context, args readmeUtgaverArgs) ([]ReadmeUtgave, error) {
	if args.First != nil && *args.First < 0 {
		return nil, errors.New("first must be a non-negative integer")
	}
	readmes, err := fetchReadmes(ctx, r.client, r.editionsURL)
	if err != nil {
		return nil, err
	}
	readmes = filterReadmes(readmes, args.Year, args.Utgave)
	if args.First != nil && int(*args.First) < len(readmes) {
		readmes = readmes[:*args.First]
	}
	return readmes, nil
}

func (r *resolver) LatestReadme(ctx context.Context) (*ReadmeUtgave, error) {
	readmes, err := fetchReadmes(ctx, r.client, r.editionsURL)
	if err != nil || len(readmes) == 0 {
		return nil, err
	}
	return &readmes[0], nil
}

// fetchReadmes fetches every edition from the editions API, newest first.
func fetchReadmes(ctx context.Context, client *http.Client, url string) ([]ReadmeUtgave, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching readme editions: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "readme-as-a-function (+https://github.com/webkom/readme-as-a-function)")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching readme editions: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching readme editions: unexpected status %s", res.Status)
	}

	var body editionsResponse
	if err := json.NewDecoder(io.LimitReader(res.Body, maxEditionsResponseBytes)).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding readme editions: %w", err)
	}
	if body.Editions == nil {
		return nil, errors.New("decoding readme editions: response has no editions")
	}

	readmes := make([]ReadmeUtgave, 0, len(body.Editions))
	for _, e := range body.Editions {
		// An edition without a year, number or page cannot be shown or linked to
		if e.EditionYear == 0 || e.EditionNumber == 0 || e.EditionURL == "" {
			continue
		}
		readmes = append(readmes, ReadmeUtgave{
			Title:  fmt.Sprintf("readme utgave nr. %d %d", e.EditionNumber, e.EditionYear),
			Image:  e.ThumbnailURL,
			Pdf:    e.EditionURL,
			Year:   e.EditionYear,
			Utgave: e.EditionNumber,
		})
	}
	sortReadmes(readmes)
	return readmes, nil
}

// sortReadmes sorts readmes newest first.
func sortReadmes(readmes []ReadmeUtgave) {
	sort.SliceStable(readmes, func(i, j int) bool {
		if readmes[i].Year == readmes[j].Year {
			return readmes[i].Utgave > readmes[j].Utgave
		}
		return readmes[i].Year > readmes[j].Year
	})
}

// filterReadmes keeps the readmes matching the given year and utgave. A nil
// filter matches everything.
func filterReadmes(readmes []ReadmeUtgave, year *int32, utgave *int32) []ReadmeUtgave {
	res := make([]ReadmeUtgave, 0, len(readmes))
	for _, r := range readmes {
		if year != nil && *year != r.Year {
			continue
		}
		if utgave != nil && *utgave != r.Utgave {
			continue
		}
		res = append(res, r)
	}
	return res
}

const graphiql = `
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

// Handle a serverless request. An empty request (GET) returns GraphiQL, any
// other request is executed as a GraphQL query.
func Handle(req []byte) string {
	return handle(req, &resolver{client: http.DefaultClient, editionsURL: editionsURL})
}

func handle(req []byte, r *resolver) string {
	if len(req) == 0 {
		return graphiql
	}

	var params struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}
	if err := json.Unmarshal(req, &params); err != nil {
		return renderError(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	schema := graphql.MustParseSchema(schemaSDL, r, graphql.UseFieldResolvers())
	response := schema.Exec(ctx, params.Query, params.OperationName, params.Variables)
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return renderError(err)
	}
	return string(responseJSON)
}

func renderError(err error) string {
	type message struct {
		Message string `json:"message"`
	}
	res, _ := json.Marshal(struct {
		Errors []message `json:"errors"`
	}{Errors: []message{{Message: err.Error()}}})
	return string(res)
}
