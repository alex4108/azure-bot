FROM alpine:3.16 AS build
RUN apk update
RUN apk upgrade
RUN apk add --update go gcc g++
WORKDIR /app
COPY ./src/*.go /app
COPY ./go.mod /app
COPY ./go.sum /app
RUN GOOS=linux go build -o ./azure-bot

FROM alpine:3.16
RUN mkdir -p /app/
COPY --from=build /app/azure-bot /app/azure-bot
RUN chmod +x /app/azure-bot
CMD ["/app/azure-bot"]