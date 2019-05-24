FROM golang AS builder
RUN go get -d github.com/navigaid/sea
RUN CGO_ENABLED=0 go build -o /bin/sea github.com/navigaid/sea

FROM alpine
WORKDIR /bin
ADD ./seashells.io ./seashells.io
COPY --from=builder /bin/sea sea
EXPOSE 1337 8000
CMD ["/bin/sea"]
