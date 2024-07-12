FROM golang

ADD mikan_spider /root
COPY ca-certificates.crt /usr/local/share/ca-certificates/
WORKDIR /root

CMD ["/root/mikan_spider"]