FROM golang

WORKDIR /app

COPY ./monitor .

RUN go mod tidy && go build -o .

EXPOSE 4080

CMD [ "./monitor" ]