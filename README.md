# readme-as-a-function

[![Go Report Card](https://goreportcard.com/badge/github.com/webkom/readme-as-a-function)](https://goreportcard.com/report/github.com/webkom/readme-as-a-function) [![Build Status](https://ci.webkom.dev/api/badges/webkom/readme-as-a-function/status.svg)](https://ci.webkom.dev/webkom/readme-as-a-function)

Graphql api as a function, runs in [openfaas](https://www.openfaas.com/).

Allows you to fetch the `n` last readmes. Also possible to filter based on year and issue number.

## Storage format

The readmes are stored in a bucket in the following format:

```sh
# The bucket has two top-level directories:
/images/
# and
/pdf/

# Both these folders follow the schema <year>/<year>-<edition>.<ext>
# So we have the following folders:
/pdf/1994/
/pdf/1995/
#...
/pdf/2020/
# The same goes for images
/images/2020/

# In all the year folders, the pdfs and images are stored:
/pdf/1994/1994-01.pdf
/pdf/1994/1994-02.pdf
# ...and so on
# Again images must follow the same schema
/images/1994/1994-01.jpg
```

> If the bucket does not follow this schema, the api will not give the correct results!

Note also that we do not check the file extension. It's assumed that all files in `/pdf/` are .pdf.
The api creates a link based on the file, so any files will resolve.
For images, any valid file format works, so jpg, png, gif, whatever.

Running on https://readme-as-a-function.abakus.no

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
$ go test -v ./...
```
