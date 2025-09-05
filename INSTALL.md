# How to Install Prequel Grafana Datasource Plugin

## 1. Prerequisites - Existing Grafana installed with Helm

Example: 
```
helm repo add grafana https://grafana.github.io/helm-charts
```
```
helm repo update
```

```
helm install my-grafana grafana/grafana \
  --namespace monitoring \
  --create-namespace
```

## 2. Build, Sign, Package and Upload

The following steps can be skipped and first official release can be downloaded from:

https://github.com/prequel-dev/prequel-grafana-datasource/releases/download/v1.0.0/prequel-prequel-datasource-1.0.0.zip


### Build

Build the plugin backend:
```
mage -v build:linux build:linuxARM64 
```

Build the plugin
```
npm run build
```

This should produce ```dist``` directory with all the files for the package.


### Self-sign the plugin (optional)

In our testing self-signing failed. It looks like Grafana removed true local dev signing in the latest sign-plugin versions. The tool now always expects a <b>GRAFANA_ACCESS_POLICY_TOKEN</b>.

```
npm install -g @grafana/sign-plugin
```

Example for locally exposed Grafana: 
```
npx @grafana/sign-plugin --rootUrls http://localhost:3000
```


### Package 

cp -r dist/ release/prequel-prequel-datasource/
cd release

zip -r prequel-prequel-datasource-1.0.0.zip prequel-prequel-datasource

### Upload
 The plugin zip needs to be uploaded somewhere, where it can be reached and downloaded from the existing k8s cluster that runs grafana. Again, is more convenient to use the official distro from GitHub: https://github.com/prequel-dev/prequel-grafana-datasource/releases/download/v1.0.0/prequel-prequel-datasource-1.0.0.zip


## 3. Install

There are few ways to install the Grafana plugin, but since the current plugin is not officially signed and published to Grafana
the easiest way is to apply the patch to the current Grafana install with Helm. 
In this example the additional init container is added that downloads the plugin into appropriate location on Grafana start.
And the `grafana.ini` is patched to allow loading of the unsigned the plugin.

#### 3.1. Create `values.yaml` file that looks like the following:

```
# values.yaml
extraInitContainers:
  - name: install-prequel-plugin
    image: busybox:1.36
    command:
      - sh
      - -c
      - |
        echo "Installing Prequel plugin..."
        mkdir -p /var/lib/grafana/plugins
        wget -O /tmp/plugin.zip https://github.com/prequel-dev/prequel-grafana-datasource/releases/download/v1.0.0/prequel-prequel-datasource-1.0.0.zip
        unzip -o /tmp/plugin.zip -d /var/lib/grafana/plugins
    volumeMounts:
      - name: plugins
        mountPath: /var/lib/grafana/plugins

# EmptyDir volume for plugins
extraVolumes:
  - name: plugins
    emptyDir: {}

# Mount the volume into the main Grafana container
extraVolumeMounts:
  - name: plugins
    mountPath: /var/lib/grafana/plugins

# Allow unsigned plugin to load
grafana.ini:
  plugins:
    allow_loading_unsigned_plugins: prequel-prequel-datasource
```

The example above uses the official releases download link. Feel free to change the script to most appropritate for your deployment.

