FROM golang

WORKDIR /app

ADD . /app

RUN ls
RUN go get
RUN go build main.go

EXPOSE 8077

CMD ["./main"]
