# readme-as-a-function

[![Go Report Card](https://goreportcard.com/badge/github.com/webkom/readme-as-a-function)](https://goreportcard.com/report/github.com/webkom/readme-as-a-function) [![Build Status](https://ci.webkom.dev/api/badges/webkom/readme-as-a-function/status.svg)](https://ci.webkom.dev/webkom/readme-as-a-function)

Graphql api as a function, runs in [openfaas](https://www.openfaas.com/).

Allows you to fetch the `n` last readmes.
Also possible to filter based on year and issue number.

Running on https://readme-as-a-function.abakus.no, used by the readme cards on the [abakus.no](https://abakus.no) frontpage.

## Data source

All editions come from the public readme API at https://readme.abakus.no/api/editions.
Every request fetches the full list of editions and filters it in memory, so new editions show up as soon as they are published on readme.abakus.no.

Each edition is mapped like this:

| GraphQL field | Editions API field | Example                                                         |
| ------------- | ------------------ | --------------------------------------------------------------- |
| `title`       | -                  | `readme utgave nr. 4 2026`                                      |
| `image`       | `thumbnailUrl`     | Cover image of the edition                                      |
| `pdf`         | `editionUrl`       | `https://readme.abakus.no/edition/2026-04`                      |
| `year`        | `editionYear`      | `2026`                                                          |
| `utgave`      | `editionNumber`    | `4`                                                             |

The `pdf` field links to the edition page on readme.abakus.no.
It keeps its name so existing clients keep working.

## To run locally

```bash
$ # Simple usage
$ go run main.go <<<  '{"query":"{latestReadme{ title }}"}' | jq
$ # As webserver at http://localhost:8000
$ go run pkg/webserver/main.go
```

### API schema

```graphql
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
```

### Testing

```bash
$ go test -v ./...
$ # Skip the test against the live editions API
$ go test -v ./... -offline
```
