FROM golang:1.15-buster

COPY collaborate-storage /
COPY resources/doc_v0_1M.dat /etc/storage.dat

RUN chmod u+x /collaborate-storage

EXPOSE 2412:2412

CMD ["/collaborate-storage server --file-path /etc/storage.dat"]
