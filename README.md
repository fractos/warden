# Warden (Go version)

Warden is an application that enables Amazon ECS containers running on an EC2 cluster to be included in the normal load-balancing of an ECS Service.

Although that sounds like perfectly normal behaviour, the nuance is in the definition of an ECS Service and how to more efficiently utilise available space in an EC2 cluster.

When the desired number of instances of a Service increases, Amazon ECS will try and place containers from a Service which have a specific port configured onto an EC2 instance in a cluster that doesn't
already have the particular Service running on it. I think this is mostly due to limitations of the AWS Elastic Load Balancing system, whereby the target instance ports must all be the same.

However, our use-case was that we wanted to have multiple similar copies of a container running as a managed Service across a cluster, configured to expect traffic on port 80 inside the container and
which did not care how they received it. The container definition essentially has a blank port setup, apart from the EXPOSE 80 line in the Dockerfile.

   