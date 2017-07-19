An pseudo implementation of a message relay server.

There is a thread to receive incoming messages and a thrad to forward the received.
Both threads cannot be brutally killed by SIGTERM (Ctrl+C) hence it must be
detected and shutdown both threads safely.