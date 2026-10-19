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
    </section>

    <section v-if="result" class="howto" aria-labelledby="howto-title">
      <h2 id="howto-title">签发后怎么用</h2>
      <ol>
        <li>
          把证书存到私钥旁边，文件名必须是「私钥名 + <code>-cert.pub</code>」。
          例如私钥是 <code>~/.ssh/id_ed25519</code>，证书就是
          <code>~/.ssh/id_ed25519-cert.pub</code>。
        </li>
        <li>
          确认证书内容和有效期：
          <pre class="cmd">ssh-keygen -L -f ~/.ssh/id_ed25519-cert.pub</pre>
          <code>Principals</code> 必须和登录用户一致，过期后无法登录。
        </li>
        <li>
          登录目标机（目标机需已信任 Vault CA）：
          <pre class="cmd">ssh -i ~/.ssh/id_ed25519 \
  -o CertificateFile=~/.ssh/id_ed25519-cert.pub \
  {{ result.principals || 'root' }}@&lt;target-host&gt;</pre>
          若证书和私钥同目录、同前缀，OpenSSH 会自动带上证书，也可直接：
          <pre class="cmd">ssh -i ~/.ssh/id_ed25519 {{ result.principals || 'root' }}@&lt;target-host&gt;</pre>
        </li>
        <li>
          可选：写进 <code>~/.ssh/config</code>，以后不用每次带参数。
          <pre class="cmd">Host &lt;alias&gt;
  HostName &lt;target-host&gt;
  User {{ result.principals || 'root' }}
  IdentityFile ~/.ssh/id_ed25519
  CertificateFile ~/.ssh/id_ed25519-cert.pub
  IdentitiesOnly yes</pre>
        </li>
      </ol>
      <p class="note">
        目标机 <code>authorized_keys</code> 不必放你的公钥。
        证书过期后重新签发即可，私钥可一直留在本机。
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
.lede {
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

.howto {
  margin-top: 28px;
  color: var(--paper);
}

.howto h2 {
  margin-bottom: 16px;
}

.howto ol {
  margin: 0;
  padding-left: 1.3rem;
  line-height: 1.65;
}

.howto li {
  margin-bottom: 16px;
}

.howto code,
.howto .cmd {
  color: var(--paper);
}

.cmd {
  margin: 8px 0 0;
  padding: 12px 14px;
  background: #0c1612;
  border-left: 3px solid var(--teal);
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 0.82rem;
}

.note {
  margin: 8px 0 0;
  color: var(--steel);
  font-size: 0.82rem;
  line-height: 1.55;
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
