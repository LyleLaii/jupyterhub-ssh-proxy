# JupyterHub SSH Proxy

An external solution allows the user ssh to the user's jupyter environment.

*This project has not been fully tested.*

## Test Environment
- k8s
- jupyterhub deployed by helm

## How to use

1. Build a notebook image with sshd, see `example/Dockerfile.example`

2. Change Helm values, use postStart to run sshd service, see `example/jupyterhub-helm-values.yaml`

3. Create a NetworkPolicy to allow proxy access ssh port

4. Create config.
```
host_key_path: ./etc/id_rsa # hostkey file path in container
jupyterhub:
  url: http://127.0.0.1:6868/hub/api # jupyterhub rest api address
  admin_token: test # jupyterhub admin token to get user pod info
  conn_user: username # user pod ssh username
  conn_passwd: password # user pod ssh password
  authorized_keys_path: '' # authorized_keys_path in user pod
  ssh_port: "2022" # user pod ssh port
  verify_tls: false
```

5. Create jupyterhub-ssh-proxy deployment

6. Create jupyterhub-ssh-proxy service

The tcp port should be used, so the TCP forwarding service should be used. NodePort is the simpleest way.

7. Create token in jupyterhub.

8. Use Jupyterhub username and token to login. Or create a authorized_keys file in user pod. 



- User should manually create `~/.bashrc` or change `/etc/bash.bashrc` when build image to allow load env via ssh
- User should manually create `~/.bash_profile` or change `/etc/profile` when build image to allow load env via ssh-command


## 
**Thanks for https://github.com/dutchcoders/sshproxy**