FROM jekyll/jekyll:4.0

RUN mkdir -p /srv/jekyll && \
    chown -R jekyll:jekyll /srv/jekyll && \
    chmod +w /srv/jekyll

WORKDIR /srv/jekyll

USER jekyll