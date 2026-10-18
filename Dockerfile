FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /task-api .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /task-api /task-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/task-api"]
