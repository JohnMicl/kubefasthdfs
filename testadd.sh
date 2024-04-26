#!/bin/bash
num_of_repetitions=1000
username=testadd
uservalue=batman
for ((id=1; id<=$num_of_repetitions; id++)); do
    # 执行curl命令
    json_data="{\"${username}_${id}\": \"${uservalue}_${id}\"}"
    curl_command="curl -XPOST 'localhost:11000/key' -d '$json_data'"
    echo $id $curl_command
    eval "$curl_command"
done
