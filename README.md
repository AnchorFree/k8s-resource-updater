# Tool to update k8s resource

This tool is for updating resources (for now its only networkpolicy) using consul-template and custom made golang script.
Tool k8s-resource-updater use Kubernetes serviceaccount to get access to networkpolicy and update it if needed.
Consul-template works as daemon and if get any updates reload k8s-resource-updater that update network policy.


### Values

- RUN_FROM_CONSUL_TEMPLATER - specify this variable if you need to update Kubernetes resource using values from consul
- CONSUL_TEMPLATER_CONFIG - config path for consul-templater
- CONSUL_ADDRESS - address to conenct to consul


### Tool Run Keys

- k8s-resource-name - name of the recource that need to update
- k8s-file-to-read - file from where nee to read values
- k8s-resource-namespace - name of namespace where resource is stored, default value is "default"
- verbose - print additional information


### How its working

1) consul-template run and generate values from template that you define in onfiguration file
2) values are reads by k8s-resource-updater from output file that generates consul-template
3) wait until gets new value from consul server and start process again


### How to use

`/k8s-resource-updater [KEYS] [COMMAND]`
where 
- KEYS - is the keys described above
- COMMAND - is 'networkpolicy' (for now)

to run specify args like this `/k8s-resource-updater --k8s-resource-name=default-allow-web-restricted --k8s-file-to-read=/output/monitoring_ipset networkpolicy` on kubernetes manifest or docker compose
This will update `default-allow-web-restricted` networkpolicy that store values in `/output/monitoring_ipset` file. File `/output/monitoring_ipset` can be generated by consul-template if you cpecify `RUN_FROM_CONSUL_TEMPLATER` environment variable, or you can add this file as volume.

### k8s-file-to-read format

file format is flat. Every value are stored as it is, and starts from new line.
Example of ip list for Kubernetes networkpolicy

```
192.168.1.1/32
192.168.1.2
10.0.0.0/8
```

### Notes

- You not necesary need to cpecify CIDR format, the tool will check if this is valid single ip and generate single ip CIDR address.
