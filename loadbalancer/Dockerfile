FROM golang:1.25

WORKDIR /app
COPY . .
RUN go mod download

RUN go build -o load_balancer .
RUN chmod +x load_balancer
CMD [ "./load_balancer" ]