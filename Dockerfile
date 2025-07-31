FROM golang:1.24 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /gk132_spb_tg2gs

FROM gcr.io/distroless/static-debian12 AS release-stage

WORKDIR /

COPY --from=build-stage /gk132_spb_tg2gs /gk132_spb_tg2gs

USER nonroot:nonroot

ENTRYPOINT ["/gk132_spb_tg2gs"]