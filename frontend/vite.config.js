import fs from 'node:fs'
import https from 'node:https'
import path from 'node:path'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const certRoot = process.env.MTLS_CERT_DIR
const clientName = process.env.MTLS_CLIENT || 'client'
const clientCert = certRoot
  ? path.join(certRoot, 'client', clientName, 'client.crt')
  : ''
const clientKey = certRoot
  ? path.join(certRoot, 'client', clientName, 'client.key')
  : ''
const caCert = certRoot ? path.join(certRoot, 'ca', 'ca.crt') : ''
const mtls =
  Boolean(certRoot) &&
  fs.existsSync(clientCert) &&
  fs.existsSync(clientKey) &&
  fs.existsSync(caCert)

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: '../build/web',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    open: true,
    proxy: {
      '/api': mtls
        ? {
            target: 'https://127.0.0.1:8080',
            changeOrigin: true,
            secure: false,
            agent: new https.Agent({
              cert: fs.readFileSync(clientCert),
              key: fs.readFileSync(clientKey),
              ca: fs.readFileSync(caCert),
            }),
          }
        : {
            target: 'http://127.0.0.1:8080',
            changeOrigin: true,
          },
    },
  },
})
