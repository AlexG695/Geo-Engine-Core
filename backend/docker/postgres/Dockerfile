FROM postgis/postgis:15-3.4

RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    make \
    cmake \
    build-essential \
    postgresql-server-dev-15 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /tmp
RUN git clone --recursive https://github.com/zachasme/h3-pg.git \
    && cd h3-pg \
    && make \
    && make install \
    && cd .. \
    && rm -rf h3-pg

RUN apt-get purge -y git make cmake build-essential postgresql-server-dev-15 \
    && apt-get autoremove -y