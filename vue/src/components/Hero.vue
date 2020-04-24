<template>
  <div class="card">
    <h3 class="close" @click="$emit('removeHero')">&times;</h3>
    <h1>{{ hero.name }}</h1>
  </div>
</template>
 
<script>
export default {
  props: ["hero"]
};
</script>

<style scoped>
.card {
  border: 1px solid black;
  margin-top: 1em;
  margin-bottom: 1em;
  width: 500px;
}
.card h3 {
  text-align: end;
  margin-right: 1em;
}
.card h3:hover {
  cursor: pointer;
}
</style>
