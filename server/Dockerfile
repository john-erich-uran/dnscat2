FROM docker.io/library/ruby:3.1.3-alpine3.17

RUN apk update && apk add make gcc musl-dev
RUN gem install bundler

COPY Gemfile /app/
COPY Gemfile.lock /app/
WORKDIR /app
RUN bundle install

COPY . /app/

EXPOSE 53/udp

CMD ["ruby", "/app/dnscat2.rb"]

# Run it
#   docker run -p 53:53/udp -it --rm mpercival/dnscat2 ruby ./dnscat2.rb foo.org
