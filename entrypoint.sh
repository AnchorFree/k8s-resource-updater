#!/usr/bin/env sh

if [ ! -z ${RUN_FROM_CONSUL_TEMPLATER} ]; then
	if [ -z ${CONSUL_ADDRESS} ]; then
		echo "CONSUL_ADDRESS is mandatory field fi RUN_FROM_CONSUL_TEMPLATER is specified"
		exit 1
	fi
        if [ -z ${CONSUL_TEMPLATER_CONFIG} ]; then
                echo "CONSUL_TEMPLATER_CONFIG is mandatory field fi RUN_FROM_CONSUL_TEMPLATER is specified"
                exit 1
        fi
	exec bin/consul-template -exec-reload-signal=SIGHUP -exec-kill-signal=SIGINT -consul-addr=${CONSUL_ADDRESS} -config=${CONSUL_TEMPLATER_CONFIG} -exec="$( echo $@ )"
else
	exec $@
fi
