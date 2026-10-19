<template>
  <main class="desk">
    <header class="mast">
      <p class="cage">Cage 157 · SSH CA</p>
      <h1>临时登录证书签发台</h1>
      <p class="lede">
        贴上本机公钥，向 Vault 领取一张短时用户证书。证书过期即失效，不必回收私钥。
      </p>
    </header>

    <section class="slip" aria-labelledby="request-title">
      <div class="slip-head">
        <h2 id="request-title">领取单</h2>
      </div>

      <label class="field">
        <span>OpenSSH 公钥</span>
        <textarea
          v-model="publicKey"
          rows="5"
          spellcheck="false"
          placeholder="ssh-ed25519 AAAA... user@host"
        />
      </label>

      <label class="field">
        <span>登录用户</span>
        <input v-model="principals" placeholder="root" />
      </label>

      <div class="actions">
        <button type="button" class="issue" :disabled="busy" @click="sign">
          {{ busy ? '正在盖章…' : '签发证书' }}
        </button>
        <p v-if="error" class="err">{{ error }}</p>
      </div>
    </section>

    <section v-if="result" class="ticket" aria-labelledby="ticket-title">
      <div class="perforation" aria-hidden="true" />
      <div class="ticket-head">
        <h2 id="ticket-title">已签发</h2>
        <p>
          序列号 {{ result.serial_number || '—' }}
          · {{ result.principals }}
        </p>
      </div>
      <pre>{{ result.signed_key }}</pre>
      <div class="ticket-actions">
        <button type="button" @click="copy(result.signed_key)">复制证书</button>
        <button type="button" @click="download(result.signed_key)">下载 .pub</button>
        <span v-if="copied" class="copied">已复制</span>
      </div>
      <p class="hint">
        保存为 <code>id_ed25519-cert.pub</code>（与私钥同目录、同前缀），然后：
        <code>ssh -i id_ed25519 -o CertificateFile=id_ed25519-cert.pub user@host</code>
      </p>
    </section>
  </main>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'

const info = reactive({
  default_principals: '',
})

const publicKey = ref('')
const principals = ref('')
const busy = ref(false)
const error = ref('')
const result = ref(null)
const copied = ref(false)

onMounted(async () => {
  try {
    const res = await fetch('/api/info')
    if (!res.ok) return
    const data = await res.json()
    Object.assign(info, data)
    if (!principals.value) principals.value = data.default_principals || 'root'
  } catch {
    /* 后端未启动时仍可编辑表单 */
  }
})

async function sign() {
  error.value = ''
  result.value = null
  copied.value = false
  busy.value = true
  try {
    const res = await fetch('/api/sign', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        public_key: publicKey.value,
        valid_principals: principals.value,
      }),
    })
    const data = await res.json()
    if (!res.ok) {
      error.value = data.error || `签发失败 (${res.status})`
      return
    }
    result.value = data
  } catch (e) {
    error.value = e.message || '无法连接签发服务'
  } finally {
    busy.value = false
  }
}

async function copy(text) {
  await navigator.clipboard.writeText(text)
  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 1600)
}

function download(text) {
  const blob = new Blob([text.endsWith('\n') ? text : text + '\n'], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'id_ed25519-cert.pub'
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<style scoped>
.desk {
  max-width: 920px;
  margin: 0 auto;
  padding: 48px 24px 80px;
}

.mast h1 {
  margin: 8px 0 12px;
  font-family: Syne, sans-serif;
  font-size: clamp(2.4rem, 6vw, 4.4rem);
  font-weight: 800;
  line-height: 0.92;
  letter-spacing: -0.04em;
}

.cage,
.lede,
.hint {
  color: var(--steel);
}

.cage {
  margin: 0;
  font-size: 0.85rem;
}

.lede {
  max-width: 42rem;
  margin: 0;
  line-height: 1.6;
}

.slip,
.ticket {
  margin-top: 36px;
  background: var(--paper);
  color: var(--ink);
  padding: 28px 28px 24px;
}

.slip-head,
.ticket-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 16px;
  border-bottom: 1px dashed var(--line);
  padding-bottom: 12px;
  margin-bottom: 20px;
}

h2 {
  margin: 0;
  font-family: Syne, sans-serif;
  font-size: 1.4rem;
  letter-spacing: -0.03em;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 16px;
}

.field span {
  color: var(--teal-ink);
  font-size: 0.82rem;
}

textarea,
input {
  width: 100%;
  border: 0;
  border-bottom: 2px solid var(--line);
  background: transparent;
  color: var(--ink);
  padding: 8px 0;
  resize: vertical;
}

textarea:focus,
input:focus,
button:focus-visible {
  outline: 2px solid var(--teal);
  outline-offset: 3px;
}

.actions {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 8px;
}

.issue {
  background: var(--stamp);
  color: #fff8f0;
  border: 0;
  padding: 12px 22px;
  font-family: Syne, sans-serif;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.issue:disabled {
  opacity: 0.55;
}

.err {
  margin: 0;
  color: var(--stamp);
}

.ticket {
  position: relative;
  background: #f3e7c8;
}

.perforation {
  position: absolute;
  inset: 0 auto 0 -10px;
  width: 20px;
  background:
    radial-gradient(circle at 10px 10px, var(--ink) 5px, transparent 5.5px) 0 0 / 20px 20px;
}

pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  line-height: 1.5;
  font-size: 0.82rem;
}

.ticket-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 16px;
}

.ticket-actions button {
  background: var(--teal);
  color: var(--paper);
  border: 0;
  padding: 8px 14px;
}

.copied {
  color: var(--teal-ink);
}

.hint {
  margin: 16px 0 0;
  font-size: 0.8rem;
  line-height: 1.55;
}

.hint code {
  color: var(--ink);
}

@media (max-width: 640px) {
  .slip-head,
  .ticket-head,
  .actions,
  .ticket-actions {
    display: grid;
    grid-template-columns: 1fr;
  }
}
</style>
