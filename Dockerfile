FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o workout-generator .

FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/workout-generator .
EXPOSE 8080
CMD ["./workout-generator"]
