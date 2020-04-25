<template>
  <div>
    <select v-model="chosenHero">
      <!-- placeholder value -->
      <option :value="null">Select a hero</option>

      <!-- available heroes -->
      <option
        v-for="hero in heroes.filter(hero => !hero.chosen)"
        :key="hero.name"
        :value="hero"
      >{{ hero.name }}</option>
    </select>
    <span>&nbsp;</span>
    <button @click="addHero(chosenHero)" :disabled="chosenHero === null">Add Hero</button>
    <span>&nbsp;</span>
    <button @click="launchMission()">Launch Mission</button>
    <br />
    <h3>Chosen Heroes</h3>
    <div class="chosen-heroes">
      <div v-for="(hero, i) in chosenHeroes" :key="hero.name">
        <strong>Slot {{ i + 1 }}:</strong>
        <Hero :hero="hero" @removeHero="removeHero(hero)" />
      </div>
    </div>
  </div>
</template>

<script>
import Hero from "./Hero";

export default {
  components: {
    Hero
  },
  props: { heroes: Array },
  data() {
    return {
      chosenHero: null,
      chosenHeroes: []
    };
  },
  methods: {
    addHero(hero) {
      if (this.chosenHeroes.length < 3) {
        this.chosenHeroes.push({ hero });
        hero.chosen = true;
        this.chosenHero = null;
      } else {
        alert("Only three heroes per mission");
      }
    },

    removeHero(hero) {
      this.chosenHeroes = this.chosenHeroes.filter(
        h => h.hero.name != hero.hero.name
      );
      hero.hero.chosen = false;
    },

    launchMission() {
      if (this.chosenHeroes.length !== 3) {
        alert("We need three heroes");
      } else {
        alert("Mission complete");
      }
    }
  }
};
</script>

<style scoped>
.chosen-heroes {
  display: flex;
  flex-flow: column;
  align-items: center;
}
</style>


