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
      <div v-for="(hero, i) in heroes.filter(hero => hero.chosen)" :key="hero.name">
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
      chosenHero: null
    };
  },
  methods: {
    addHero(hero) {
      const chosenHeroes = this.heroes.filter(hero => hero.chosen).length;
      if (chosenHeroes < 3) {
        hero.chosen = true;
      } else {
        alert("Only three heroes per mission");
      }
      this.chosenHero = null;
    },

    removeHero(hero) {
      hero.chosen = false;
    },

    launchMission() {
      const chosenHeroes = this.heroes.filter(hero => hero.chosen).length;
      if (chosenHeroes !== 3) {
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


