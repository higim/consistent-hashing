FROM golang:1.25 AS builder

WORKDIR /app
COPY . .
RUN go mod download

RUN go build -o node .
CMD [ "./node" ]