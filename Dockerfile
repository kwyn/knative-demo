# Pre build step build binary
FROM golang:alpine AS build-env
ADD /src /src
RUN cd /src && go build -o goapp

# Mount binary into alpine container
FROM alpine
WORKDIR /app
COPY --from=build-env /src/goapp /app/
ENTRYPOINT ./goapp