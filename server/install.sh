#!/usr/bin/env bash

set -ex

[ `whoami` = root ] || exec sudo su -c $0 root

sudo yum -y update
sudo yum -y install make gcc

if ! command -v rvm &> /dev/null
then
    echo "install rvm"
    curl -sSL https://get.rvm.io | bash
    . ~/.bashrc
fi

source /usr/local/rvm/scripts/rvm

rvm install ruby-3.1.3
rvm use ruby-3.1.3

cd /home/ec2-user/server
gem install bundler
bundle install
