# React Micro Frontend Server Go

A Server for React Micro Frontends, process frontend SPA request and metadata API.

## Features

* Response frontend page's request at any route of SPA.
* Metadata API: get info, install a new version of micro frontend or uninstall it dynamically.
* Custom site config: HTML template, user default route.
* Serve static files. It is a easy way to deploy our micro frontends on laptop.
* Link preload headers. We can use server push (HTTP/2) with nginx `http2_push_preload on`.
* A/B testing control.
