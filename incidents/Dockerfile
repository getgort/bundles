FROM golang:1.19

COPY . /bundle
WORKDIR /bundle
RUN go build -o bookmark
RUN chmod +x bookmark

ENTRYPOINT [ "/bundle/bookmark" ]
