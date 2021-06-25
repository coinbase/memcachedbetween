# build
FROM golang:alpine AS build-env
RUN apk --no-cache add build-base git gcc
RUN git clone https://github.com/coinbase/memcachedbetween.git
RUN cd memcachedbetween && go build -o memcachedbetween

## app
FROM alpine
WORKDIR /app
COPY --from=build-env /go/memcachedbetween/memcachedbetween /app/
CMD ./memcachedbetween