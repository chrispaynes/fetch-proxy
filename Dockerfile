FROM golang:1.10.0 AS build_stage
WORKDIR /go/src/fetch-proxy/
COPY . .
RUN export GOBIN="/go/bin" \
    && go get ./... \
    && CGO_ENABLED=0 GOOS=linux go install ./main.go

FROM scratch
COPY --from=build_stage /go/bin/main .
ENTRYPOINT ["./main"]