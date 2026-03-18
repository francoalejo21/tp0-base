#!/bin/bash
MESSAGE="Hello world"
NETWORK="tp0_testing_net"
SERVER_HOST="server"
PORT="12345"

RECIBIDO=$( echo $MESSAGE | docker run --rm -i --network $NETWORK busybox nc $SERVER_HOST $PORT)
if [ "$RECIBIDO" = "$MESSAGE" ]; then
    echo "action: test_echo_server | result: success"
else
    echo "action: test_echo_server | result: fail"
fi

#Si queremos eliminar la imagen despues de validar que funciona el server:
# docker rmi busybox