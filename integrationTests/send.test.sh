set -e

# create a new queue
# create a file with JSON messages
# use send to put the messages into the queue
# validate the number of messages is the approximate amount of messages in the queue (wait for 1 min)
# clean up - delete queue