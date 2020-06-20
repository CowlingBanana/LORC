# LORC
LORC - Low Orbit RECON Cannon

The idea of LORC is pretty simple, to speed up recon gathering by distributing the work. I came to this idea while looking at ways to quickly processs MILLIONS of IPs through my recon workflow. Doing it on one machine, even if that machine was powerful, would take days. But, if I distribute the workload to many machines I can start reducing that time window drastically. Thus, LORC was born! 

LORC gets it's name from the seminal DDOS tool Low Orbit Ion Cannon. You could also think of this in a more friendly way as being like SETI but instead of hunting for aliens we are hunting for software bugs!

The design of LORC is fairly simple. A LORC server is stood up somewhere that LORC Clients can reach, then a LORC Client is run on a machine to add it to the work pool. When the Client connects, the Server send a request asking what recon tools the Client has available. Once the Client responds with its capabilities the Server will begin sending jobs that the Client can handle. Once the Client executes the given job, it sends the results back to the Server for storage and viewing.

My grand goal here is to setup a public LORC Server and give free access to the data collected by LORC Clients. My hope is that those who find this project install the LORC Client to improve the collection speed and help gather more recon data, benefiting themselves, and the community as a whole.
