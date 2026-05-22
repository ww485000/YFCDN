FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
RUN go build -o /out/yfcdn-control ./cmd/control && go build -o /out/yfcdn-agent ./cmd/agent

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/yfcdn-control /usr/local/bin/yfcdn-control
COPY --from=build /out/yfcdn-agent /usr/local/bin/yfcdn-agent
COPY web ./web
RUN mkdir -p /app/data
ENV YFCDN_ADDR=:8080
ENV YFCDN_DATA=/app/data/yfcdn.json
ENV YFCDN_STATIC=/app/web
EXPOSE 8080
CMD ["yfcdn-control"]
