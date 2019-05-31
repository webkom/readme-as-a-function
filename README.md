# readme-as-a-function

Graphql api as a function, runs in [openfaas](https://www.openfaas.com/).

Allows you to fetch the `n` last readmes from https://readme.abakus.no. Also possible to filter based on year and issue number.

Running on https://readme-as-a-function.abakus.no

## To run locally

```bash
$ dep ensure
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
```

### Testing

```bash
$ go test
```
