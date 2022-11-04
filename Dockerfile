FROM golang AS Builder

ENV GOPROXY=https://goproxy.cn,direct

COPY . /app

WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main

FROM alpine as Runner

COPY --from=Builder /app/main /app/main

WORKDIR /app

ENV PATH="${PATH}:/app"

ENTRYPOINT ["main"]