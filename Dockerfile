FROM golang:1.15.5-alpine3.12 AS builder
WORKDIR /src/
ADD . /src/
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags='-extldflags=-static' -o /bin/main 

FROM gcr.io/distroless/static
COPY --from=builder /bin/main /bin/main

ENTRYPOINT [ "/bin/main" ]