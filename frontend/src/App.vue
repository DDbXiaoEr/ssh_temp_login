<template>
  <main class="hall">
    <header class="plate">
      <h1>临时登录证书签发台</h1>
      <p class="lede">
        贴上本机公钥，领取一张短时用户证书。过期即失效，不必回收私钥。
      </p>
    </header>

    <form class="kiosk" aria-labelledby="request-title" @submit.prevent="sign">
      <h2 id="request-title" class="sr">申请证书</h2>

      <div class="field key">
        <div class="field-head">
          <label for="pubkey">OpenSSH 公钥</label>
          <label class="pick">
            选择文件
            <input type="file" accept=".pub,text/plain" @change="loadKeyFile" />
          </label>
        </div>
        <textarea
          id="pubkey"
          v-model="publicKey"
          rows="7"
          spellcheck="false"
          placeholder="ssh-ed25519 AAAA... user@host"
        />
      </div>

      <div class="side">
        <label class="field">
          <span>登录用户</span>
          <input v-model="principals" placeholder="root" autocomplete="username" />
        </label>

        <div class="actions">
          <button type="submit" class="issue" :disabled="busy">
            {{ busy ? '正在签发' : '签发证书' }}
          </button>
          <p v-if="error" class="err" role="alert">{{ error }}</p>
        </div>
      </div>
    </form>

    <section v-if="result" class="pass" aria-labelledby="pass-title">
      <div class="stripe" aria-hidden="true" />
      <div class="pass-face">
        <div class="pass-meta">
          <h2 id="pass-title">{{ result.principals || '—' }}</h2>
          <p>序列 {{ result.serial_number || '—' }}</p>
        </div>
        <pre>{{ result.signed_key }}</pre>
        <div class="pass-actions">
          <button type="button" @click="copy(result.signed_key)">
            {{ copied ? '已复制' : '复制证书' }}
          </button>
          <button type="button" @click="download(result.signed_key)">下载证书</button>
        </div>
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

async function loadKeyFile(event) {
  const input = event.target
  const file = input.files && input.files[0]
  input.value = ''
  if (!file) return
  error.value = ''
  try {
    const text = (await file.text()).trim()
    if (!text) {
      error.value = '文件是空的'
      return
    }
    publicKey.value = text
  } catch {
    error.value = '无法读取所选文件'
  }
}

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
.hall {
  max-width: 960px;
  margin: 0 auto;
  padding: 40px 24px 88px;
}

.plate h1,
.pass-meta h2,
input {
  font-family: "Big Shoulders Display", "Impact", "IBM Plex Sans", sans-serif;
}

.plate h1 {
  margin: 0 0 12px;
  color: var(--enamel);
  font-size: clamp(2.4rem, 6vw, 4.2rem);
  font-weight: 800;
  letter-spacing: -0.04em;
  line-height: 0.92;
}

.pass-meta h2 {
  font-weight: 800;
  letter-spacing: -0.04em;
  line-height: 0.86;
}

.lede {
  max-width: 38rem;
  margin: 0;
  color: var(--mute);
  line-height: 1.55;
}

.kiosk {
  display: grid;
  grid-template-columns: minmax(0, 1.6fr) minmax(16rem, 0.9fr);
  gap: 28px 36px;
  margin-top: 40px;
  padding: 28px 0 8px;
  border-top: 8px solid var(--enamel);
}

.sr {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
}

.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field span,
.field-head,
.field-head label {
  color: var(--mute);
  font-size: 0.92rem;
}

.field-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}

.pick {
  position: relative;
  color: var(--enamel);
  font-size: 0.88rem;
  cursor: pointer;
}

.field-head .pick {
  color: var(--enamel);
}

.pick input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
  font-size: 0;
  padding: 0;
}

textarea,
input {
  width: 100%;
  border: 0;
  border-radius: 0;
  background: var(--badge);
  color: var(--ink);
  padding: 12px 14px;
}

textarea {
  min-height: 11.5rem;
  resize: vertical;
  font-family: "IBM Plex Mono", ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.84rem;
  line-height: 1.5;
}

input {
  font-size: 1.35rem;
  font-weight: 700;
  letter-spacing: -0.03em;
}

textarea:focus,
input:focus,
button:focus-visible {
  outline: 3px solid var(--enamel);
  outline-offset: 3px;
}

.side {
  display: flex;
  flex-direction: column;
  gap: 22px;
}

.actions {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
}

.issue {
  background: var(--sodium);
  color: var(--sodium-ink);
  border: 0;
  padding: 14px 22px;
  font-weight: 600;
}

.issue:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.err {
  margin: 0;
  color: var(--fault);
  line-height: 1.4;
}

.pass {
  margin-top: 36px;
  background: var(--badge);
  color: var(--ink);
  box-shadow: 12px 12px 0 var(--enamel-deep);
}

.stripe {
  height: 42px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.12), transparent 40%),
    repeating-linear-gradient(
      90deg,
      var(--stripe) 0 3px,
      #1a140e 3px 5px,
      #2e241a 5px 8px
    );
}

.pass-face {
  padding: 22px 24px 20px;
}

.pass-meta {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 16px;
  margin-bottom: 16px;
}

.pass-meta h2 {
  margin: 0;
  font-size: clamp(2.2rem, 6vw, 3.6rem);
}

.pass-meta p {
  margin: 0;
  color: var(--mute);
  font-family: "IBM Plex Mono", ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.84rem;
}

pre {
  margin: 0;
  padding: 14px 16px;
  background: var(--slot);
  white-space: pre-wrap;
  word-break: break-all;
  line-height: 1.5;
  font-family: "IBM Plex Mono", ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.8rem;
}

.pass-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 16px;
}

.pass-actions button {
  background: var(--enamel);
  color: var(--badge);
  border: 0;
  padding: 9px 14px;
}

.howto {
  margin-top: 40px;
  max-width: 42rem;
}

.howto h2 {
  margin: 0 0 16px;
  font-size: 1.2rem;
}

.howto ol {
  margin: 0;
  padding-left: 1.25rem;
  line-height: 1.65;
}

.howto li {
  margin-bottom: 16px;
}

.howto code,
.cmd {
  font-family: "IBM Plex Mono", ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.84em;
}

.cmd {
  margin: 8px 0 0;
  padding: 12px 14px;
  background: var(--enamel-deep);
  color: var(--badge);
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 0.8rem;
}

.note {
  margin: 4px 0 0;
  color: var(--mute);
  font-size: 0.92rem;
  line-height: 1.55;
}

@media (max-width: 760px) {
  .kiosk,
  .pass-meta {
    display: grid;
    grid-template-columns: 1fr;
  }
}

@media (prefers-reduced-motion: no-preference) {
  .pass {
    animation: print 0.35s ease-out;
  }
}

@keyframes print {
  from {
    opacity: 0;
    transform: translateY(-12px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}
</style>
