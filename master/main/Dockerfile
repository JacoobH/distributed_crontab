FROM golang:latest
RUN mkdir -p /home/work/crontab/master
WORKDIR /home/work/crontab/master
COPY main main
COPY master.json master.json
COPY webroot/ webroot/
RUN chmod +x main
EXPOSE 8080
ENTRYPOINT ["./main"]