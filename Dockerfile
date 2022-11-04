FROM golang AS builder

WORKDIR /

FROM alpine

WORKDIR /app

ENV PATH="${PATH}:/usr/local/go/bin:/go/bin"
ENV GOROOT="/usr/local/go"
ENV GOPATH="/go"

RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

COPY --from=builder /usr/local/go /usr/local/go

ENTRYPOINT ["go", "run"]


