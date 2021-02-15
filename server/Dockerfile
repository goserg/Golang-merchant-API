FROM golang

RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN go get github.com/lib/pq
RUN go get github.com/360EntSecGroup-Skylar/excelize
RUN go build -o main .
CMD ["/app/main"]
