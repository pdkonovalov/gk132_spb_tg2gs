FROM golang:1.24 AS build-stage

WORKDIR /go/src/app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 go build -o /go/bin/app

RUN mkdir /go/bin/data

FROM gcr.io/distroless/static-debian12

COPY --from=build-stage /go/bin/app /

COPY --from=build-stage --chown=:nonroot /go/bin/data /

CMD ["/app"]