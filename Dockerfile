FROM alpine

ADD mikan_spider /root

WORKDIR /root

ENTRYPOINT ["/root/mikan_spider"]