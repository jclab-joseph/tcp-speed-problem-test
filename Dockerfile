FROM golang:1.24-alpine3.21 as builder

RUN mkdir -p /build/
COPY . /build/

RUN cd /build/ && \
    ls -al && \
    CGO_ENABLED=0 go build -o /build/server.exe ./

FROM alpine:3.21
COPY --from=builder /build/server.exe /server.exe

EXPOSE 3000
CMD /server.exe
